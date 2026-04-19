package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

// UserAccountConsumer 用户账户创建事件消费者（Saga 协调器）
type UserAccountConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	store   UserAccountStoreInterface
}

// UserAccountStoreInterface 用户账户操作接口（使用 sqlc 类型）
type UserAccountStoreInterface interface {
	// CreateUserAccount 创建用户账户
	CreateUserAccount(ctx context.Context, arg CreateUserAccountParams) (int64, error)
	// UpdateUserStatus 更新用户状态
	UpdateUserStatus(ctx context.Context, arg UpdateUserStatusParams) error
	// CheckEventProcessed 检查事件是否已处理
	CheckEventProcessed(ctx context.Context, requestID string) (bool, error)
	// MarkEventProcessed 标记事件已处理
	MarkEventProcessed(ctx context.Context, arg MarkEventProcessedParams) error
}

// CreateUserAccountParams 创建账户参数
type CreateUserAccountParams struct {
	Owner    string `json:"owner"`
	Currency string `json:"currency"`
}

// UpdateUserStatusParams 更新状态参数
type UpdateUserStatusParams struct {
	Status       sql.NullString `json:"status"`
	RequestID    sql.NullString `json:"request_id"`
	FailedReason sql.NullString `json:"failed_reason"`
	Username     string         `json:"username"`
}

// MarkEventProcessedParams 标记事件参数
type MarkEventProcessedParams struct {
	RequestID string          `json:"request_id"`
	Username  string          `json:"username"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

func NewUserAccountConsumer(amqpURL string, store UserAccountStoreInterface) (*UserAccountConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开 Channel 失败：%w", err)
	}

	// QoS 限制
	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("设置 QoS 失败：%w", err)
	}

	return &UserAccountConsumer{
		conn:    conn,
		channel: ch,
		store:   store,
	}, nil
}

// Start 启动消费者
func (c *UserAccountConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		UserProfileQueue,
		"simple_bank_user_profile_worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("注册消费者失败：%w", err)
	}

	log.Println("RabbitMQ 用户账户消费者已启动，正在监听用户资料创建队列...")

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

// processMessage 处理用户资料创建消息
func (c *UserAccountConsumer) processMessage(msg amqp.Delivery) {
	var payload ProfileCreatedEvent
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("消息格式解析失败：%v\n", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("收到用户资料创建消息 -> 用户：%s, success=%v, request_id=%s\n",
		payload.Username, payload.Success, payload.RequestId)

	// 幂等性检查
	processed, err := c.store.CheckEventProcessed(context.Background(), payload.RequestId)
	if err != nil {
		log.Printf("检查事件处理状态失败：%v，重新入队\n", err)
		msg.Nack(false, true)
		return
	}
	if processed {
		log.Printf("事件已处理过，跳过：%s\n", payload.RequestId)
		msg.Ack(false)
		return
	}

	// 只有成功时才创建账户
	if !payload.Success {
		log.Printf("用户资料创建失败，执行补偿事务\n")
		// 更新用户状态为 FAILED
		c.store.UpdateUserStatus(context.Background(), UpdateUserStatusParams{
			Status:    sql.NullString{String: "FAILED", Valid: true},
			RequestID: sql.NullString{String: payload.RequestId, Valid: true},
			FailedReason: sql.NullString{String: payload.ErrorMessage, Valid: payload.ErrorMessage != ""},
			Username:  payload.Username,
		})
		// 标记事件已处理
		c.store.MarkEventProcessed(context.Background(), MarkEventProcessedParams{
			RequestID: payload.RequestId,
			Username:  payload.Username,
			EventType: "profile_failed",
			Payload:   nil,
		})
		msg.Ack(false)
		return
	}

	// 创建默认账户
	accountId, err := c.createUserAccount(context.Background(), payload.Username)
	if err != nil {
		log.Printf("创建用户账户失败：%v\n", err)
		// 更新用户状态为 PARTIALLY_INITIALIZED
		c.store.UpdateUserStatus(context.Background(), UpdateUserStatusParams{
			Status:    sql.NullString{String: "PARTIALLY_INITIALIZED", Valid: true},
			RequestID: sql.NullString{String: payload.RequestId, Valid: true},
			Username:  payload.Username,
		})
		// 标记事件已处理
		c.store.MarkEventProcessed(context.Background(), MarkEventProcessedParams{
			RequestID: payload.RequestId,
			Username:  payload.Username,
			EventType: "account_failed",
			Payload:   nil,
		})
		msg.Ack(false)
		return
	}

	// 更新用户状态为 INITIALIZED
	err = c.store.UpdateUserStatus(context.Background(), UpdateUserStatusParams{
		Status:    sql.NullString{String: "INITIALIZED", Valid: true},
		RequestID: sql.NullString{String: payload.RequestId, Valid: true},
		Username:  payload.Username,
	})
	if err != nil {
		log.Printf("更新用户状态失败：%v\n", err)
	}

	// 标记事件已处理
	c.store.MarkEventProcessed(context.Background(), MarkEventProcessedParams{
		RequestID: payload.RequestId,
		Username:  payload.Username,
		EventType: "account_created",
		Payload:   json.RawMessage(fmt.Sprintf(`{"account_id": %d}`, accountId)),
	})

	log.Printf("用户初始化完成 -> 用户：%s, 账户 ID: %d\n", payload.Username, accountId)
	msg.Ack(false)
}

// createUserAccount 创建用户默认账户
func (c *UserAccountConsumer) createUserAccount(ctx context.Context, username string) (int64, error) {
	// 创建默认 CNY 账户
	accountId, err := c.store.CreateUserAccount(ctx, CreateUserAccountParams{
		Owner:    username,
		Currency: "CNY",
	})
	if err != nil {
		return 0, fmt.Errorf("创建账户失败：%w", err)
	}

	return accountId, nil
}
