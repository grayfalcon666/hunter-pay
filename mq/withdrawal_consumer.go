package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/smartwalle/alipay/v3"
	"github.com/streadway/amqp"
)

// ErrIdempotencyKeyAlreadyExists is returned by the bank client when an idempotency
// key has already been used (meaning the operation was already applied).
var ErrIdempotencyKeyAlreadyExists = errors.New("idempotency key already exists")

// BankClient 接口，用于调用 simple-bank
type BankClient interface {
	Transfer(ctx context.Context, fromAccount, toAccount int64, amount int64, idempotencyKey string, tradeType ...string) error
	Unfreeze(ctx context.Context, accountID, amount int64, idempotencyKey string) error
	WithdrawFromFrozen(ctx context.Context, accountID, amount int64, idempotencyKey string, description string) error
}

type WithdrawalConsumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	store        withdrawalStore
	alipayClient *alipay.Client
	bankClient   BankClient
	platformID   int64 // 平台账户 ID (999999)
}

type withdrawalStore interface {
	UpdateWithdrawalStatus(ctx context.Context, outBizNo, status, remark string) error
	UpdateWithdrawalSuccess(ctx context.Context, outBizNo, payFundOrderId string) error
	TryClaimWithdrawal(ctx context.Context, outBizNo string) (int64, error)
}

func NewWithdrawalConsumer(amqpURL string, store withdrawalStore, alipayClient *alipay.Client, bankClient BankClient, platformID int64) (*WithdrawalConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	ch.Qos(1, 0, false)

	return &WithdrawalConsumer{
		conn:         conn,
		channel:      ch,
		store:        store,
		alipayClient: alipayClient,
		bankClient:   bankClient,
		platformID:   platformID,
	}, nil
}

func (c *WithdrawalConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		WithdrawalQueue,
		"payment_withdrawal_worker",
		false, // 手动 Ack
		false, false, false, nil,
	)
	if err != nil {
		return err
	}

	log.Println("支付网关消费者已启动，正在监听提现打款队列...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.channel.Close()
				c.conn.Close()
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				c.processWithdrawal(msg)
			}
		}
	}()
	return nil
}

func (c *WithdrawalConsumer) processWithdrawal(msg amqp.Delivery) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var payload WithdrawalMessage
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("提现消息解析失败: %v\n", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("开始处理提现单: %s, 支付宝账号: %s, 金额: %d\n", payload.OutBizNo, payload.AlipayAccount, payload.Amount)

	// Step 1: 乐观锁检查，防止重复处理
	// 只有从 PROCESSING 状态才能继续；如果已是 REFUNDING/SUCCESS/FAILED，说明重复消费
	rowsAffected, err := c.store.TryClaimWithdrawal(ctx, payload.OutBizNo)
	if err != nil {
		log.Printf("乐观锁更新失败，重新入队: outBizNo=%s, err=%v\n", payload.OutBizNo, err)
		c.nackWithBackoff(msg, false, true)
		return
	}
	if rowsAffected == 0 {
		// 状态已被推进，说明之前已经处理过，直接 Ack
		log.Printf("提现单已处理过，跳过: outBizNo=%s\n", payload.OutBizNo)
		msg.Ack(false)
		return
	}

	// Step 2: 调用支付宝打款
	var p = alipay.FundTransUniTransfer{}
	p.OutBizNo = payload.OutBizNo
	p.TransAmount = fmt.Sprintf("%.2f", float64(payload.Amount)/100)
	p.ProductCode = "TRANS_ACCOUNT_NO_PWD"
	p.BizScene = "DIRECT_TRANSFER"
	p.PayeeInfo = &alipay.PayeeInfo{
		Identity:     payload.AlipayAccount,
		IdentityType: "ALIPAY_LOGON_ID",
		Name:         payload.AlipayRealName,
	}

	rsp, err := c.alipayClient.FundTransUniTransfer(ctx, p)

	if err != nil {
		log.Printf("调用支付宝接口网络异常，重新入队等待重试: %v\n", err)
		c.nackWithBackoff(msg, false, true)
		return
	}

	if rsp.Code != "10000" || rsp.Status == "FAIL" {
		log.Printf("支付宝拒绝打款: %s - %s\n", rsp.Msg, rsp.SubMsg)

		// Step 3: 执行 Unfreeze 补偿（将用户 frozen_balance 解冻回 available）
		refundIdempotencyKey := "refund_" + payload.OutBizNo
		unfreezeErr := c.bankClient.Unfreeze(ctx, payload.AccountID, payload.Amount, refundIdempotencyKey)

		if unfreezeErr != nil {
			// 检查是否是幂等性冲突（已经退过款了）
			if isIdempotencyError(unfreezeErr) {
				log.Printf("提现退款已执行过（幂等拦截），outBizNo=%s\n", payload.OutBizNo)
			} else {
				log.Printf("提现 Unfreeze 补偿失败，outBizNo=%s, err=%v\n", payload.OutBizNo, unfreezeErr)
				// Unfreeze 失败，重新入队等待重试
				c.nackWithBackoff(msg, false, true)
				return
			}
		}

		errorDetail := rsp.SubMsg
		if errorDetail == "" {
			errorDetail = rsp.Msg
		}
		c.store.UpdateWithdrawalStatus(context.Background(), payload.OutBizNo, "FAILED", errorDetail)

		log.Printf("Saga 补偿完成，资金已从 frozen_balance 解冻回用户账户。outBizNo=%s\n", payload.OutBizNo)
		msg.Ack(false)
		return
	}

	// Step 4: 支付宝打款成功，永久从 frozen_balance 扣款
	withdrawIdempotencyKey := "withdraw_" + payload.OutBizNo
	withdrawErr := c.bankClient.WithdrawFromFrozen(ctx, payload.AccountID, payload.Amount, withdrawIdempotencyKey, "提现")
	if withdrawErr != nil {
		// 检查是否是幂等性冲突（已经扣过款了）
		if isIdempotencyError(withdrawErr) {
			log.Printf("提现扣款已执行过（幂等拦截），outBizNo=%s\n", payload.OutBizNo)
		} else {
			// 严重错误！支付宝已打款成功，但扣款失败，需要人工处理
			log.Printf("严重错误！支付宝打款成功但扣款失败，outBizNo=%s, err=%v\n", payload.OutBizNo, withdrawErr)
			c.store.UpdateWithdrawalStatus(context.Background(), payload.OutBizNo, "FAILED",
				fmt.Sprintf("支付宝打款成功但系统扣款失败，需要人工处理: %v", withdrawErr))
			msg.Ack(false)
			return
		}
	}

	fundOrderId := rsp.PayFundOrderId
	if fundOrderId == "" {
		fundOrderId = rsp.OrderId
	}

	c.store.UpdateWithdrawalSuccess(context.Background(), payload.OutBizNo, fundOrderId)

	log.Printf("提现打款成功！支付宝流水号: %s\n", fundOrderId)
	msg.Ack(false)
}

func (c *WithdrawalConsumer) nackWithBackoff(msg amqp.Delivery, multiple bool, requeue bool) {
	time.Sleep(2 * time.Second)
	msg.Nack(multiple, requeue)
}

// isIdempotencyError 检查 gRPC 错误是否是幂等性键冲突
func isIdempotencyError(err error) bool {
	if err == nil {
		return false
	}
	// gRPC 错误会包装原始错误，检查错误消息或 status code
	errStr := err.Error()
	return strings.Contains(errStr, "idempotency") ||
		strings.Contains(errStr, "AlreadyExists") ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "ALREADY_EXISTS")
}
