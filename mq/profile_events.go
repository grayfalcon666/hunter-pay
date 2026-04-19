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
	// 复用 user-profile-service 的交换机配置
	BountyExchange          = "user_exchange"
	ProfileUpdateQueue      = "profile_update_queue"
	ProfileUpdateRoutingKey = "profile.update"
	ProfileUpdateDLX        = "profile_update_dlx"
	ProfileUpdateDLQ        = "profile_update_dlq"

	// 履约指数重算队列
	FulfillmentRecalcQueue      = "fulfillment_recalc_queue"
	FulfillmentRecalcRoutingKey  = "fulfillment.recalc"
	FulfillmentRecalcDLX         = "fulfillment_recalc_dlx"
	FulfillmentRecalcDLQ         = "fulfillment_recalc_dlq"
)

// ProfileUpdateEvent 用户画像更新事件
type ProfileUpdateEvent struct {
	Username                  string `json:"username"`
	BountyID                  int64  `json:"bounty_id"`
	DeltaCompleted            int32  `json:"delta_completed"`
	DeltaEarnings             int64  `json:"delta_earnings"`
	DeltaPosted               int32  `json:"delta_posted"`
	DeltaCompletedAsEmployer  int32  `json:"delta_completed_as_employer"`
	RequestID                 string `json:"request_id"`
}

// FulfillmentRecalcEvent 履约指数重算事件
type FulfillmentRecalcEvent struct {
	Username string `json:"username"`
	Role     string `json:"role"` // "HUNTER" or "EMPLOYER"
	BountyID int64  `json:"bounty_id"`
	RequestID string `json:"request_id"`
}

// ProfileUpdateProducer 用户画像更新事件生产者
type ProfileUpdateProducer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewProfileUpdateProducer(amqpURL string) (*ProfileUpdateProducer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开 Channel 失败：%w", err)
	}

	// 开启 Publisher Confirms（异步模式，不需要等待确认）
	err = ch.Confirm(false)
	if err != nil {
		log.Printf("警告: 开启 Publisher Confirms 失败：%v，将使用无确认模式\n", err)
	}

	// 声明死信交换机
	err = ch.ExchangeDeclare(ProfileUpdateDLX, "direct", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明 DLX 失败：%w", err)
	}

	// 声明死信队列
	_, err = ch.QueueDeclare(ProfileUpdateDLQ, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明 DLQ 失败：%w", err)
	}

	// 绑定死信队列到死信交换机
	err = ch.QueueBind(ProfileUpdateDLQ, ProfileUpdateDLQ, ProfileUpdateDLX, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 DLQ 失败：%w", err)
	}

	// 声明主交换机（复用 user-profile-service 的 user_exchange）
	err = ch.ExchangeDeclare(BountyExchange, "direct", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明主 Exchange 失败：%w", err)
	}

	// 声明主队列（配置死信）
	args := amqp.Table{
		"x-dead-letter-exchange": ProfileUpdateDLX,
	}

	_, err = ch.QueueDeclare(ProfileUpdateQueue, true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("声明 Queue %s 失败：%w", ProfileUpdateQueue, err)
	}

	// 绑定主队列到交换机
	err = ch.QueueBind(ProfileUpdateQueue, ProfileUpdateRoutingKey, BountyExchange, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 Queue 失败：%w", err)
	}

	log.Println("RabbitMQ ProfileUpdateProducer 初始化成功")
	return &ProfileUpdateProducer{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish 发布用户画像更新事件（无确认模式）
func (p *ProfileUpdateProducer) Publish(ctx context.Context, event *ProfileUpdateEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	err = p.channel.Publish(
		BountyExchange,
		ProfileUpdateRoutingKey,
		true,  // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			DeliveryMode:  amqp.Persistent,
			CorrelationId: event.RequestID,
			Timestamp:     time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	log.Printf("ProfileUpdateEvent 已发布：username=%s, bounty_id=%d, request_id=%s\n",
		event.Username, event.BountyID, event.RequestID)
	return nil
}

// Close 关闭连接
func (p *ProfileUpdateProducer) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

// FulfillmentRecalcProducer 履约指数重算事件生产者
type FulfillmentRecalcProducer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewFulfillmentRecalcProducer(amqpURL string) (*FulfillmentRecalcProducer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开 Channel 失败：%w", err)
	}

	err = ch.Confirm(false)
	if err != nil {
		log.Printf("警告: 开启 Publisher Confirms 失败：%v\n", err)
	}

	// 声明死信交换机
	err = ch.ExchangeDeclare(FulfillmentRecalcDLX, "direct", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明 DLX 失败：%w", err)
	}

	// 声明死信队列
	_, err = ch.QueueDeclare(FulfillmentRecalcDLQ, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明 DLQ 失败：%w", err)
	}

	err = ch.QueueBind(FulfillmentRecalcDLQ, FulfillmentRecalcDLQ, FulfillmentRecalcDLX, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 DLQ 失败：%w", err)
	}

	// 声明主交换机
	err = ch.ExchangeDeclare(BountyExchange, "direct", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("声明主 Exchange 失败：%w", err)
	}

	// 声明主队列（配置死信）
	args := amqp.Table{
		"x-dead-letter-exchange": FulfillmentRecalcDLX,
	}
	_, err = ch.QueueDeclare(FulfillmentRecalcQueue, true, false, false, false, args)
	if err != nil {
		return nil, fmt.Errorf("声明 Queue %s 失败：%w", FulfillmentRecalcQueue, err)
	}

	err = ch.QueueBind(FulfillmentRecalcQueue, FulfillmentRecalcRoutingKey, BountyExchange, false, nil)
	if err != nil {
		return nil, fmt.Errorf("绑定 Queue 失败：%w", err)
	}

	log.Println("RabbitMQ FulfillmentRecalcProducer 初始化成功")
	return &FulfillmentRecalcProducer{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish 发布履约指数重算事件
func (p *FulfillmentRecalcProducer) Publish(ctx context.Context, event *FulfillmentRecalcEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败：%w", err)
	}

	err = p.channel.Publish(
		BountyExchange,
		FulfillmentRecalcRoutingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			DeliveryMode:  amqp.Persistent,
			CorrelationId: event.RequestID,
			Timestamp:     time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败：%w", err)
	}

	log.Printf("FulfillmentRecalcEvent 已发布：username=%s, role=%s, bounty_id=%d\n",
		event.Username, event.Role, event.BountyID)
	return nil
}

// Close 关闭连接
func (p *FulfillmentRecalcProducer) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
