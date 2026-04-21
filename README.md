# Escrow — 悬赏担保平台

[![Go](https://img.shields.io/badge/Go-1.21-blue)](https://go.dev)
[![Vue.js](https://img.shields.io/badge/Vue.js-3-blue)](https://vuejs.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue)](https://postgresql.org)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-3.12-blue)](https://rabbitmq.com)
[![gRPC](https://img.shields.io/badge/gRPC-enabled-brightgreen)](https://grpc.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

一个基于 Go 微服务 + Vue 3 的悬赏担保平台，支持悬赏发布、接单、聊天、支付（支付宝）、履约评价的完整业务流程。

---

## 架构总览

```
Browser ──► Gateway :8080 ──┬──► SimpleBank  :11452 (HTTP) / :11453 (gRPC)
                             ├──► Escrow-Bounty :8087 (HTTP) / :9097 (gRPC)
                             ├──► UserProfile   :8088 (HTTP) / :9098 (gRPC)
                             └──► PaymentService :8082 (HTTP) / :8083 (gRPC)

Browser ──► Gateway /ws ────► Escrow-Bounty :9099 (WebSocket)

Alipay ──► Gateway /webhook/alipay ──► PaymentService :8082

PaymentService :8083 ──gRPC──► SimpleBank :11453
Escrow-Bounty  :9097 ──gRPC──► SimpleBank :11453
UserProfile    :9098 ──gRPC──► Escrow-Bounty :9097
```

**共享基础设施**: RabbitMQ (5672), Redis (6379), PostgreSQL (5432/5433/5434)

---

## 服务说明

### Gateway
API 网关，统一对外暴露 HTTP 端口（8080），反向代理到各后端服务，并将 gRPC 调用转换为 JSON over HTTP（gRPC-gateway）。同时处理 WebSocket 升级路由和支付宝回调。

### SimpleBank
账户与账本服务，基于 [simplebank](https://github.com/techschool/simplebank) 项目改造。支持账户创建、转账（Transfer）、冻结/解冻（Freeze/Unfreeze）、从冻结余额扣款（WithdrawFromFrozen）等原子操作，提供幂等性键保护。资金流转的最终仲裁者。

### Escrow-Bounty
悬赏核心业务服务，管理悬赏生命周期（发布 → 接单 → 进行中 → 提交 → 完成/取消）、评论与聊天、实时 WebSocket 推送。是 MQ 事件的主要生产者。

### UserProfile
用户画像服务，管理用户档案、评价、履约指数计算。消费 Escrow-Bounty 发来的 MQ 事件，维护用户的统计数据（总收入、完成数、好评率）和履约指数（牛顿冷却衰减模型）。

### PaymentService
支付服务，对接支付宝（当面付/转账到支付宝账户）。用户充值后通过支付宝异步回调通知，调用 MQ 通知 SimpleBank 账本更新。提现走 Saga 补偿模式：MQ 消费端调用支付宝打款，失败则补偿解冻冻结余额。

---

## 消息队列（RabbitMQ）设计

### Escrow-Bounty → UserProfile

Escrow-Bounty 是事件的**生产者**，通过两个独立的 Outbox Worker 轮询数据库表发布事件到 RabbitMQ，不依赖本地 MQ 事务保证可靠性。

**Outbox 模式**：业务数据和待发送事件写入同一数据库事务，Outbox Worker 异步扫描 `PENDING` 记录发布到 MQ，发布成功后标记 `COMPLETED`，失败重试。

```
Escrow-Bounty DB
  ├── profile_update_outbox   (ProfileUpdateEvent)
  └── fulfillment_outbox       (FulfillmentRecalcEvent)
       ↓ 轮询扫描
  RabbitMQ: user_exchange (direct)
       ├── profile_update_queue      → UserProfile 消费
       └── fulfillment_recalc_queue  → UserProfile 消费
```

**Exchange/Queue 拓扑**:
- `user_exchange` (direct) — 主交换机
- `profile_update_queue` — 绑定 `profile.update`，DLX 为 `profile_update_dlx`
- `fulfillment_recalc_queue` — 绑定 `fulfillment.recalc`，DLX 为 `fulfillment_recalc_dlx`
- 所有队列均配置 DLQ（死信队列）接收处理失败的消息

**ProfileUpdateEvent** 字段：
```json
{ "username", "bounty_id", "delta_completed", "delta_earnings",
  "delta_posted", "delta_completed_as_employer", "request_id" }
```
用于 UserProfile 累计更新用户的统计数据（总收入、悬赏完成数等）。

**FulfillmentRecalcEvent** 字段：
```json
{ "username", "role": "HUNTER|EMPLOYER", "bounty_id", "request_id" }
```
触发 UserProfile 重新计算该用户在特定悬赏上的履约评分。

### PaymentService → SimpleBank

```
PaymentService DB: payments / withdrawals
     ↓ 同步（HTTP 回调后同步发布）
  RabbitMQ: payment_exchange (direct)
       ├── payment_queue       → SimpleBank 消费（充值入账）
       └── withdrawal_queue   → PaymentService 自己消费（提现打款）
```

**PaymentSuccessMessage**（充值成功）：
```json
{ "username", "account_id", "amount", "out_trade_no" }
```
SimpleBank 消费后调用 `Transfer` 将资金从平台账户转入用户账户。

**WithdrawalMessage**（提现请求）：
```json
{ "account_id", "amount", "alipay_account", "alipay_real_name",
  "out_biz_no" }
```
PaymentService 自己消费，先调支付宝打款，失败则 Unfreeze 补偿。

---

## 悬赏状态机与 Saga

### 状态定义

```
PAYING ──► PENDING ──► IN_PROGRESS ──► SUBMITTED ──► COMPLETED
                │              │
                └── CANCELED ──┘         └── FAILED
```

- **PAYING**: 悬赏创建成功，资金冻结中（中间状态，对用户不可见）
- **PENDING**: 资金已冻结，悬赏在广场可见，等待猎人接单
- **IN_PROGRESS**: 有猎人接单且被雇主确认，开始工作
- **SUBMITTED**: 猎人提交工作成果，等待雇主审核
- **COMPLETED**: 雇主审核通过，资金从平台打款给猎人
- **CANCELED**: 雇主在 PENDING 阶段取消，资金退回
- **FAILED**: 系统异常（如打款失败后人工处理状态）

### CreateBounty（发布悬赏）

```
1. DB: bounty.status = PAYING，写入数据库
2. gRPC SimpleBank.Freeze: 冻结雇主账户金额（幂等键: publish_bounty_{id}）
   - 成功 → 步骤3
   - 失败 → bounty.status = FAILED，返回错误
3. DB: bounty.status = PENDING
4. DB: 写入 profile_update_outbox（delta_posted +1）
5. 返回 bounty 给前端
```

### CompleteBounty / ApproveBounty（完成悬赏）

两种路径最终都走向COMPLETED：
- **CompleteBounty**: 雇主直接宣布完成（简单路径）
- **ApproveBounty**: 猎人先 Submit，雇主 Approve（审核路径）

```
1. DB: bounty.status = COMPLETED（乐观锁，WHERE status IN ('IN_PROGRESS','SETTLING')）
2. DB: 写入 profile_update_outbox（猎人: delta_completed +1, delta_earnings；雇主: delta_completed_as_employer +1）
3. DB: 写入 fulfillment_outbox（触发履约指数重算，默认 rating=3）
4. gRPC SimpleBank.BountyPayout: 从雇主账户转账给猎人（幂等键: complete_bounty_{id}）
   - 幂等冲突 → 已打款过，直接返回成功
   - 失败 → bounty.status 回滚为 IN_PROGRESS，允许重试
```

### CancelBounty（取消悬赏）

```
1. DB: bounty.status = CANCELED（乐观锁，WHERE status = 'PENDING'）
2. DB: 所有申请标记为 REJECTED
3. DB: 写入 TaskRecord（outcome=NEUTRAL）
4. gRPC SimpleBank.Unfreeze: 解冻雇主资金（幂等键: cancel_bounty_refund_{id}）
   - 幂等冲突 → 已退款，直接返回成功
   - 失败 → bounty.status 回滚为 PENDING，允许重试
```

**幂等性**：所有跨服务调用（Freeze/Unfreeze/Transfer/BountyPayout）均使用 `idempotency_key = {operation}_{bounty_id}` 保护，SimpleBank 层面对重复请求返回成功而非报错。

---

## 数据库

| 服务 | 数据库 | 端口 |
|------|--------|------|
| SimpleBank | simple_bank | 5432 |
| Escrow-Bounty | escrow_db | 5433 |
| UserProfile | escrow_db | 5433 |
| PaymentService | payment_db | 5434 |

**金额约定**：所有货币金额以**分**为单位（BIGINT），前端除以 100 展示。

---

## 履约指数

UserProfile 维护两个履约指数：`hunter_fulfillment_index`（猎人视角）和 `employer_fulfillment_index`（雇主视角），范围 0~100，基于牛顿冷却模型：

```
新分数 = 旧分数 * exp(-距离/半衰期) + 本次评分 * (1 - exp(-距离/半衰期))
```

每当悬赏完成或取消，Escrow-Bounty 写入 `fulfillment_outbox`，UserProfile 的 `FulfillmentOutboxWorker` 消费后重新计算。评分在用户提交评价后才最终确定（SettleTaskRecordRating），未评价时使用默认 rating=3。

---

## 实时通信（WebSocket）

Escrow-Bounty 内置 `wshub` 处理两类 WebSocket 连接：

- **悬赏聊天室** (`/ws`): 同一悬赏的雇主和猎人共享一个对话，所有成员广播
- **私信**: 用户与用户之间的一对一私信

Hub 使用内存 `map[convID]map[*Client]bool` 追踪所有连接者，注册/注销通过 Channel 异步处理，消息持久化到 `escrow_db.chat_messages` 表。

---

## 支付宝接入

### 充值（用户 → 平台）

```
1. 用户前端发起充值请求 → PaymentService.CreatePayment
2. PaymentService 创建 payment 记录（status=PROCESSING）
3. 调用支付宝当面付获取支付二维码链接
4. 返回支付链接给前端
5. 用户扫码支付
6. 支付宝异步回调 POST /webhook/alipay → PaymentService
7. 验签通过 → 更新 payment.status=SUCCESS
8. 发布 PaymentSuccessMessage 到 RabbitMQ
9. SimpleBank 消费 → Transfer 从平台账户转充值金额到用户账户
```

PaymentService 启动时通过 `checkDependencies` 确保 `/webhook/alipay` 可达（回调 URL 必须是公网可访问），不可达时拒绝创建充值订单。

### 提现（平台 → 用户支付宝）

```
1. 用户前端发起提现请求 → PaymentService.CreateWithdrawal
2. PaymentService 创建 withdrawal 记录（status=PROCESSING）
3. 发布 WithdrawalMessage 到 RabbitMQ withdrawal_queue
4. PaymentService 消费消息：
   a. 调支付宝 FundTransUniTransfer 打款
   b. 成功 → WithdrawFromFrozen 永久扣款 → status=SUCCESS
   c. 失败 → Unfreeze 补偿解冻 → status=FAILED
```

### Webhook 公开访问

由于支付宝需要回调公网 `/webhook/alipay`，网关通过环境变量 `WEBHOOK_BASE_URL` 配置公网隧道地址。开发环境使用 Cloudflare Tunnel 或 ngrok 等工具建立临时公网隧道。

---

## 前端

Vue 3 + Quasar 2 + Pinia + Axios。所有 API 调用经过 Gateway（`/api/v1/*`），JWT 存储在 `localStorage`。

**主要页面**：
- `/` — 悬赏大厅（列表）
- `/bounty/:id` — 悬赏详情（含评论区、申请人列表）
- `/hunters` — 猎人招募广场
- `/profile/:username` — 用户资料
- `/reviews/:username` — 用户评价
- `/my/tasks` — 我的任务
- `/my/bounties` — 我发布的悬赏
- `/my/comments` — 我的评论

---

## 环境变量

参考 `.env.example`，主要分组：

```bash
# SimpleBank
DB_SOURCE, HTTP/GPRC_SERVER_ADDRESS, RABBITMQ_URL, REDIS_ADDRESS, TOKEN_SYMMETRIC_KEY

# Gateway
GATEWAY_PORT, SIMPLEBANK/ESCROW_BOUNTY/USER_PROFILE/PAYMENT_SERVICE_URL

# Escrow-Bounty
DB_SOURCE, HTTP/GPRC_SERVER_ADDRESS, SIMPLE_BANK_ADDRESS,
PLATFORM_ESCROW_ACCOUNT_ID, USER_PROFILE_SERVICE_ADDRESS, WS_SERVER_ADDRESS

# UserProfile
DB_SOURCE, HTTP/GPRC_SERVER_ADDRESS, ESCROW_BOUNTY_ADDRESS

# PaymentService
DB_SOURCE, HTTP/GPRC_SERVER_ADDRESS, SIMPLE_BANK_ADDRESS,
PLATFORM_ESCROW_ACCOUNT_ID, GATEWAY_URL,
APP_ID, APP_PRIVATE_KEY, ALIPAY_PUBLIC_KEY,
WEBHOOK_BASE_URL, FRONTEND_BASE_URL
```

---

## 快速启动

```bash
# 基础设施
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=123456 postgres:16
docker run -d -p 5672:5672 -p 15672:15672 rabbitmq:3-management
docker run -d -p 6379:6379 redis:7

# 后端（各开一个终端）
cd simplebank           && make server
cd escrow-bounty       && make server
cd user-profile-service && make server
cd payment-service     && make server
cd gateway             && go run gateway/main.go

# 前端
cd escrow-web && yarn dev
```

**公网隧道**（支付宝回调）：
```bash
cloudflared tunnel --url http://localhost:8080
# 设置 WEBHOOK_BASE_URL=https://xxx.xxx.cloudflaremqtt.net
```
