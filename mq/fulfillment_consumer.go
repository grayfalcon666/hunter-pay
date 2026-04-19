package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/grayfalcon666/user-profile-service/db"
	"github.com/streadway/amqp"
)

const (
	FulfillmentRecalcQueue      = "fulfillment_recalc_queue"
	FulfillmentRecalcRoutingKey  = "fulfillment.recalc"
	BountyExchange              = "user_exchange"
)

// FulfillmentRecalcEvent 履约指数重算事件
type FulfillmentRecalcEvent struct {
	Username  string `json:"username"`
	Role      string `json:"role"`
	BountyID  int64  `json:"bounty_id"`
	RequestID string `json:"request_id"`
}

// FulfillmentConsumer 履约指数重算事件消费者
type FulfillmentConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	store   db.Store
}

func NewFulfillmentConsumer(amqpURL string, store db.Store) (*FulfillmentConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败：%w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开 Channel 失败：%w", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("设置 QoS 失败：%w", err)
	}

	return &FulfillmentConsumer{
		conn:    conn,
		channel: ch,
		store:   store,
	}, nil
}

func (c *FulfillmentConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		FulfillmentRecalcQueue,
		"user_profile_service_fulfillment_worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("注册消费者失败：%w", err)
	}

	log.Println("RabbitMQ FulfillmentConsumer 已启动，正在监听履约重算队列...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("收到上下文取消信号，FulfillmentConsumer 正在退出...")
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

func (c *FulfillmentConsumer) processMessage(msg amqp.Delivery) {
	var payload FulfillmentRecalcEvent
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("履约重算消息格式解析失败：%v\n", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("收到履约重算事件 -> username=%s, role=%s, bounty_id=%d, request_id=%s\n",
		payload.Username, payload.Role, payload.BountyID, payload.RequestID)

	score, err := c.store.RecalculateFulfillmentIndex(context.Background(), payload.Username, payload.Role)
	if err != nil {
		log.Printf("履约指数重算失败 [username=%s, role=%s]: %v\n", payload.Username, payload.Role, err)
		msg.Nack(false, true)
		return
	}

	log.Printf("履约指数重算成功：username=%s, role=%s, new_score=%d\n", payload.Username, payload.Role, score)
	msg.Ack(false)
}

func (c *FulfillmentConsumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
