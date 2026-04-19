package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

const (
	// 用户初始化相关配置
	UserExchange       = "user_exchange"
	UserCreatedQueue   = "user_created_queue"
	UserProfileQueue   = "user_profile_queue"
	UserAccountQueue   = "user_account_queue"
	UserInitDLX        = "user_init_dlx"
	UserInitDLQ        = "user_init_dlq"

	UserCreatedRoutingKey = "user.created"
	UserProfileRoutingKey = "user.profile.created"
	UserAccountRoutingKey = "user.account.created"
)

type UserProducer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewUserProducer(amqpURL string) (*UserProducer, error) {
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

	_, err = ch.QueueDeclare(UserCreatedQueue, true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("声明 UserCreated Queue 失败：%w", err)
	}
	err = ch.QueueBind(UserCreatedQueue, UserCreatedRoutingKey, UserExchange, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 UserCreated Queue 失败：%w", err)
	}

	_, err = ch.QueueDeclare(UserProfileQueue, true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("声明 UserProfile Queue 失败：%w", err)
	}
	err = ch.QueueBind(UserProfileQueue, UserProfileRoutingKey, UserExchange, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 UserProfile Queue 失败：%w", err)
	}

	_, err = ch.QueueDeclare(UserAccountQueue, true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("声明 UserAccount Queue 失败：%w", err)
	}
	err = ch.QueueBind(UserAccountQueue, UserAccountRoutingKey, UserExchange, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 UserAccount Queue 失败：%w", err)
	}

	log.Println("RabbitMQ 用户初始化生产者初始化成功")
	return &UserProducer{
		conn:    conn,
		channel: ch,
	}, nil
}

// PublishUserCreatedEvent 发布用户创建事件
func (p *UserProducer) PublishUserCreatedEvent(ctx context.Context, event *UserCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err = p.channel.Publish(
		UserExchange,
		UserCreatedRoutingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			CorrelationId: event.RequestId, // 用于追踪
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	// 等待 ACK
	select {
	case confirmed := <-confirms:
		if confirmed.Ack {
			log.Printf("用户创建事件已发布：%s\n", event.Username)
			return nil
		}
		return fmt.Errorf("RabbitMQ 拒绝了该消息 (Nack)")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("等待 RabbitMQ 确认超时")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishProfileCreatedEvent 发布用户资料创建事件
func (p *UserProducer) PublishProfileCreatedEvent(ctx context.Context, event *ProfileCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	routingKey := UserProfileRoutingKey
	if !event.Success {
		routingKey = UserAccountRoutingKey // 失败时发送到账户队列进行补偿
	}

	err = p.channel.Publish(
		UserExchange,
		routingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			CorrelationId: event.RequestId,
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	select {
	case confirmed := <-confirms:
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

// PublishAccountCreatedEvent 发布账户创建事件
func (p *UserProducer) PublishAccountCreatedEvent(ctx context.Context, event *AccountCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err = p.channel.Publish(
		UserExchange,
		UserAccountRoutingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			CorrelationId: event.RequestId,
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	select {
	case confirmed := <-confirms:
		if confirmed.Ack {
			log.Printf("账户创建事件已发布：%s (account_id=%d, success=%v)\n", event.Username, event.AccountId, event.Success)
			return nil
		}
		return fmt.Errorf("RabbitMQ 拒绝了该消息 (Nack)")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("等待 RabbitMQ 确认超时")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *UserProducer) Close() {
	p.channel.Close()
	p.conn.Close()
}
