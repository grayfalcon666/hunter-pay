# CLAUDE.md

A **Bounty Escrow Platform** — 4 Go microservices + 1 API gateway + 1 Vue.js frontend.

## Architecture

```
Browser ──► Gateway :8080 ──┬──► SimpleBank :11452
                            ├──► Escrow-Bounty :8087
                            ├──► UserProfile :8088
                            └──► PaymentService :8082

Browser ──► Gateway /ws ────► Escrow-Bounty :9099 (WebSocket)
Browser ──► Gateway /webhook/alipay ──► PaymentService :8082

PaymentService :8083 ──gRPC──► SimpleBank :11453
EscrowBounty :9097 ───gRPC──► SimpleBank :11453
```

| Service | Module | HTTP | gRPC | WS | Database |
|---------|--------|------|------|-----|---------|
| SimpleBank | `github.com/grayfalcon666/simplebank` | 11452 | 11453 | — | `:5432` simple_bank |
| Escrow-Bounty | `github.com/grayfalcon666/escrow-bounty` | 8087 | 9097 | 9099 | `:5433` escrow_db |
| UserProfile | `github.com/grayfalcon666/user-profile-service` | 8088 | 9098 | — | `:5433` escrow_db |
| PaymentService | `github.com/grayfalcon666/payment-service` | 8082 | 8083 | — | `:5434` payment_db |
| Gateway | `github.com/grayfalcon666/gateway` | 8080 | — | — | — |

Shared: **RabbitMQ** (5672), **Redis** (6379).

## Gateway Routing (port 8080)

```
/api/v1/auth/*         → SimpleBank :11452
/api/v1/account/*      → SimpleBank :11452
/api/v1/transfers/*    → SimpleBank :11452
/api/v1/bounties/*     → Escrow-Bounty :8087
/api/v1/profiles/*     → UserProfile :8088
/api/v1/users/*        → UserProfile :8088
/api/v1/reviews/*      → UserProfile :8088
/api/v1/payments/*     → PaymentService :8082
/api/v1/withdrawals*   → PaymentService :8082
/api/v1/conversations* → Escrow-Bounty :8087
/api/v1/comments/*     → Escrow-Bounty :8087
/ws                    → Escrow-Bounty :9099 (WebSocket)
/webhook/alipay        → PaymentService :8082 (no JWT)
/swagger/              → Swagger UI
/health                 → 200 OK
```

## Bounty State Machine

```
PAYING → PENDING → IN_PROGRESS → COMPLETED
                 ↘ CANCELED (refund via compensating transaction)
                 ↘ FAILED
```

Saga pattern for CreateBounty, CompleteBounty, CancelBounty.

## Running Services

```bash
# Infrastructure
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=123456 postgres:16
docker run -d -p 5672:5672 -p 15672:15672 rabbitmq:3-management
docker run -d -p 6379:6379 redis:7

# Backend services
cd simplebank           && make server
cd escrow-bounty       && make server
cd user-profile-service && make server
cd payment-service     && make server
cd gateway             && go run gateway/main.go

# Frontend
cd escrow-web && yarn dev
```

## Public Webhook URL (for Alipay callbacks)

Alipay needs to reach your `/webhook/alipay` endpoint from the public internet.

Any tunneling solution works — examples:

```bash
# Cloudflare Tunnel
cloudflared tunnel --url http://localhost:8080

# ngrok
ngrok http 8080

# frp,花生棒, etc.
```

Set `WEBHOOK_BASE_URL` to your public tunnel URL (e.g. `https://xxx.ngrok.io`).

PaymentService checks webhook reachability before allowing充值 — if the URL is unreachable, `CreatePayment` fails with an error. No vendor lock-in.

## Code Generation

```bash
make sqlc     # Generate DB code from .sql files
make proto    # Generate Go code from .proto files
make mockgen  # Generate gomock interfaces
```

After modifying `.proto` files: `make proto` in all affected services, then `gateway/scripts/copy_swagger.sh`.

## Key Conventions

- **Amounts in cents (BIGINT)** — divide by 100 for display
- **JWT secret**: `12345678901234567890123456789012` (dev only)
- **Platform escrow account ID**: `1`
- gRPC-gateway: JSON bodies use camelCase
- Idempotency keys for transfers

## Plan Mode Rule
必须在计划最后写好明确的任务清单

## Frontend (escrow-web)

Vue 3 + Quasar 2 + Pinia + Axios. API calls go through `/api/v1`. JWT stored in `localStorage` as `token` and `username`. use yarn.
