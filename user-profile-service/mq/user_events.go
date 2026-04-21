package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

const (
	// 复用 SimpleBank 的配置
	UserExchange     = "user_exchange"
	UserCreatedQueue = "user_created_queue"
	UserProfileQueue = "user_profile_queue"
	UserAccountQueue = "user_account_queue"
	UserInitDLX      = "user_init_dlx"
	UserInitDLQ      = "user_init_dlq"

	UserCreatedRoutingKey = "user.created"
	UserProfileRoutingKey = "user.profile.created"
	UserAccountRoutingKey = "user.account.created"

	// ProfileUpdateQueue from escrow-bounty
	ProfileUpdateQueue      = "profile_update_queue"
	ProfileUpdateRoutingKey = "profile.update"
	ProfileUpdateDLX        = "profile_update_dlx"
	ProfileUpdateDLQ        = "profile_update_dlq"
)

// UserEventProducer 用户事件生产者
type UserEventProducer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	confirms chan amqp.Confirmation // 全局唯一的确认通道
	mu       sync.Mutex
}

func NewUserEventProducer(amqpURL string) (*UserEventProducer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开 Channel 失败：%w", err)
	}

	// 开启 Publisher Confirms
	err = ch.Confirm(false)
	if err != nil {
		return nil, fmt.Errorf("开启 Publisher Confirms 失败：%w", err)
	}

	// 声明死信交换机
	err = ch.ExchangeDeclare(UserInitDLX, "direct", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明 DLX 失败：%w", err)
	}

	// 声明死信队列
	_, err = ch.QueueDeclare(UserInitDLQ, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明 DLQ 失败：%w", err)
	}

	// 绑定死信队列到死信交换机
	err = ch.QueueBind(UserInitDLQ, UserInitDLQ, UserInitDLX, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 DLQ 失败：%w", err)
	}

	// 声明主交换机
	err = ch.ExchangeDeclare(UserExchange, "direct", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明主 Exchange 失败：%w", err)
	}

	// 声明主队列（配置死信）
	args := amqp.Table{
		"x-dead-letter-exchange": UserInitDLX,
	}

	for _, queue := range []string{UserCreatedQueue, UserProfileQueue, UserAccountQueue} {
		_, err = ch.QueueDeclare(queue, true, false, false, false, args)
		if err != nil {
			return nil, fmt.Errorf("声明 Queue %s 失败：%w", queue, err)
		}
	}

	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	log.Println("RabbitMQ 用户事件生产者初始化成功")
	return &UserEventProducer{
		conn:     conn,
		channel:  ch,
		confirms: confirms,
	}, nil
}

// UserCreatedEvent 用户创建事件
type UserCreatedEvent struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	RequestId string    `json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ProfileCreatedEvent 用户资料创建事件
type ProfileCreatedEvent struct {
	Username     string `json:"username"`
	RequestId    string `json:"request_id"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// AccountCreatedEvent 账户创建事件
type AccountCreatedEvent struct {
	Username     string `json:"username"`
	AccountId    int64  `json:"account_id"`
	RequestId    string `json:"request_id"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// PublishProfileCreatedEvent 发布用户资料创建事件
func (p *UserEventProducer) PublishProfileCreatedEvent(ctx context.Context, event *ProfileCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	routingKey := UserProfileRoutingKey
	if !event.Success {
		routingKey = UserAccountRoutingKey
	}

	err = p.channel.Publish(
		UserExchange,
		routingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			DeliveryMode:  amqp.Persistent,
			CorrelationId: event.RequestId,
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	select {
	case confirmed := <-p.confirms:
		if confirmed.Ack {
			log.Printf("用户资料创建事件已发布：%s (success=%v)\n", event.Username, event.Success)
			return nil
		}
		return fmt.Errorf("RabbitMQ 拒绝了该消息 (Nack)")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("等待 RabbitMQ 确认超时")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *UserEventProducer) Close() {
	p.channel.Close()
	p.conn.Close()
}

// ProfileUpdateEvent 用户画像更新事件（来自 escrow-bounty）
type ProfileUpdateEvent struct {
	Username                 string `json:"username"`
	BountyID                 int64  `json:"bounty_id"`
	DeltaCompleted           int32  `json:"delta_completed"`
	DeltaEarnings            int64  `json:"delta_earnings"`
	DeltaPosted              int32  `json:"delta_posted"`
	DeltaCompletedAsEmployer int32  `json:"delta_completed_as_employer"`
	RequestID                string `json:"request_id"`
}

// PublishFulfillmentRecalcEvent publishes a fulfillment recalculation event via RabbitMQ.
// FulfillmentRecalcEvent is defined in fulfillment_consumer.go for sharing between producer and consumer.
func (p *UserEventProducer) PublishFulfillmentRecalcEvent(ctx context.Context, event *FulfillmentRecalcEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	err = p.channel.Publish(
		UserExchange,
		FulfillmentRecalcRoutingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			DeliveryMode:  amqp.Persistent,
			CorrelationId: event.RequestID,
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	select {
	case confirmed := <-p.confirms:
		if confirmed.Ack {
			log.Printf("履约重算事件已发布：username=%s, role=%s\n", event.Username, event.Role)
			return nil
		}
		return fmt.Errorf("RabbitMQ 拒绝了该消息 (Nack)")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("等待 RabbitMQ 确认超时")
	case <-ctx.Done():
		return ctx.Err()
	}
}
