package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	db "simplebank/db/sqlc"
	"strings"

	"github.com/streadway/amqp"
)

const PaymentQueue = "payment_success_queue"

type PaymentSuccessMessage struct {
	Username   string `json:"username"`
	AccountID  int64  `json:"account_id"`
	Amount     int64  `json:"amount"`
	OutTradeNo string `json:"out_trade_no"`
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	store   db.Store
}

func NewConsumer(amqpURL string, store db.Store) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("连接rabiitmq失败: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开channel失败: %w", err)
	}

	// 限制每次只拉取 1 条消息 防止多个 Consumer 时出现分配不均
	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("设置 QoS 失败: %w", err)
	}

	// 声明队列（幂等操作，多次声明不影响），需与 payment-service 声明参数一致
	args := amqp.Table{
		"x-dead-letter-exchange": "payment_dlx",
	}
	_, err = ch.QueueDeclare(
		PaymentQueue, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		args,        // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("声明队列失败: %w", err)
	}

	return &Consumer{
		conn:    conn,
		channel: ch,
		store:   store,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		PaymentQueue,
		"simple_bank_payment_worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("注册消费者失败: %w", err)
	}

	log.Println("RabbitMQ 消费者已启动，正在监听支付成功队列...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("收到上下文取消信号，消费者正在退出...")
				c.channel.Close()
				c.conn.Close()
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Println("RabbitMQ Channel 已关闭")
					return
				}
				c.processMessage(msg)
			}
		}
	}()

	return nil
}

func (c *Consumer) processMessage(msg amqp.Delivery) {
	var payload PaymentSuccessMessage
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("消息格式解析失败，进入死信队列: %v\n", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("收到充值消息 -> 用户: %s, 账户: %d, 金额: %d, 流水号: %s\n",
		payload.Username, payload.AccountID, payload.Amount, payload.OutTradeNo)

	// 越权自保
	account, err := c.store.GetAccount(context.Background(), payload.AccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("账号 %d 不存在！消息进入死信队列\n", payload.AccountID)
			msg.Nack(false, false) // 致命错误，踢入死信队列
			return
		}
		log.Printf("数据库查询抖动，重新入队重试: %v\n", err)
		msg.Nack(false, true) // 网络抖动，重试
		return
	}

	if account.Owner != payload.Username {
		log.Printf("越权警告！拦截到用户 [%s] 试图给 [%s] 的账户 [%d] 充值！进入死信队\n",
			payload.Username, account.Owner, payload.AccountID)
		msg.Nack(false, false)
		return
	}

	// 幂等性事务执行 (Idempotency Anti-Replay)
	err = c.store.AddBalanceTx(context.Background(), db.AddBalanceTxParams{
		AccountID:      payload.AccountID,
		Amount:         payload.Amount,
		IdempotencyKey: payload.OutTradeNo, // 唯一单号做防重放键
	})

	if err != nil {
		// 检查是否是 PostgreSQL 的唯一约束冲突 (重复键异常)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") ||
			strings.Contains(err.Error(), "SQLSTATE 23505") {
			log.Printf("ℹ订单 %s 已经入账过，幂等性拦截生效，丢弃消息。\n", payload.OutTradeNo)
			msg.Ack(false) // 已经处理过的消息，直接当做处理成功告诉 MQ
			return
		}

		log.Printf("事务执行失败，重新入队重试: %v\n", err)
		msg.Nack(false, true)
		return
	}

	log.Printf("✅ 入账成功！账户 [%d] 余额安全增加 %d\n", payload.AccountID, payload.Amount)
	msg.Ack(false)
}
