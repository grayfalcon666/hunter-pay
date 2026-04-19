package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/grayfalcon666/user-profile-service/db"
	"github.com/grayfalcon666/user-profile-service/models"
	"github.com/streadway/amqp"
)

// UserEventConsumer 用户事件消费者
type UserEventConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	producer *UserEventProducer
	store    db.Store
}

func NewUserEventConsumer(amqpURL string, store db.Store, producer *UserEventProducer) (*UserEventConsumer, error) {
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

	return &UserEventConsumer{
		conn:     conn,
		channel:  ch,
		producer: producer,
		store:    store,
	}, nil
}

// Start 启动消费者
func (c *UserEventConsumer) Start(ctx context.Context) error {
	// 消费用户创建事件
	msgs, err := c.channel.Consume(
		UserCreatedQueue,
		"user_profile_service_worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("注册消费者失败：%w", err)
	}

	log.Println("RabbitMQ 用户事件消费者已启动，正在监听用户创建队列...")

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

	// 启动第二个消费者处理 ProfileUpdateQueue
	if err := c.startProfileUpdateConsumer(ctx); err != nil {
		return err
	}

	return nil
}

// startProfileUpdateConsumer 启动 ProfileUpdateQueue 的消费者
func (c *UserEventConsumer) startProfileUpdateConsumer(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		ProfileUpdateQueue,
		"user_profile_service_profile_update_worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("注册 ProfileUpdateQueue 消费者失败：%w", err)
	}

	log.Println("RabbitMQ ProfileUpdateQueue 消费者已启动，正在监听用户画像更新队列...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("收到上下文取消信号，ProfileUpdate 消费者正在退出...")
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Println("ProfileUpdateQueue Channel 已关闭")
					return
				}
				c.processProfileUpdateMessage(msg)
			}
		}
	}()

	return nil
}

// processMessage 处理用户创建消息
func (c *UserEventConsumer) processMessage(msg amqp.Delivery) {
	var payload UserCreatedEvent
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("消息格式解析失败：%v\n", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("收到用户创建事件 -> 用户：%s, email=%s\n", payload.Username, payload.Email)

	// 先检查是否已存在
	existingProfile, err := c.store.GetProfile(context.Background(), payload.Username)
	if err == nil && existingProfile != nil {
		// 资料已存在，直接发布成功事件
		log.Printf("用户资料已存在：%s\n", payload.Username)
		event := &ProfileCreatedEvent{
			Username:  payload.Username,
			RequestId: payload.RequestId,
			Success:   true,
		}
		if err := c.producer.PublishProfileCreatedEvent(context.Background(), event); err != nil {
			log.Printf("发布 ProfileCreatedEvent 失败：%v\n", err)
			msg.Nack(false, true)
			return
		}
		msg.Ack(false)
		return
	}

	// 直接使用 store 创建用户资料
	profile, err := c.store.CreateProfile(context.Background(), payload.Username, &models.CreateProfileParams{
		ExpectedSalaryMin: "",
		ExpectedSalaryMax: "",
		WorkLocation:      "",
		ExperienceLevel:    models.ExperienceEntry,
		Bio:               "",
		AvatarURL:         "",
	})

	var event *ProfileCreatedEvent
	if err != nil {
		// 创建失败，发布失败事件
		log.Printf("创建用户资料失败：%v\n", err)
		event = &ProfileCreatedEvent{
			Username:     payload.Username,
			RequestId:    payload.RequestId,
			Success:      false,
			ErrorMessage: err.Error(),
		}
	} else {
		// 创建成功，更新 initialization_request_id
		log.Printf("用户资料创建成功：%s\n", profile.Username)

		// 更新 initialization_request_id（忽略错误，因为资料已创建成功）
		c.store.UpdateProfileInitRequestID(context.Background(), payload.Username, payload.RequestId)

		event = &ProfileCreatedEvent{
			Username:  payload.Username,
			RequestId: payload.RequestId,
			Success:   true,
		}
	}

	ctx := context.Background()
	if err := c.producer.PublishProfileCreatedEvent(ctx, event); err != nil {
		log.Printf("发布 ProfileCreatedEvent 失败：%v\n", err)
		msg.Nack(false, true) // 重试
		return
	}

	msg.Ack(false)
}

// processProfileUpdateMessage 处理用户画像更新消息
func (c *UserEventConsumer) processProfileUpdateMessage(msg amqp.Delivery) {
	var payload ProfileUpdateEvent
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("ProfileUpdate 消息格式解析失败：%v\n", err)
		msg.Nack(false, false) // 不重试，消息格式错误
		return
	}

	log.Printf("收到用户画像更新事件 -> 用户：%s, bounty_id=%d, delta_completed=%d, delta_earnings=%d, delta_posted=%d, delta_completed_as_employer=%d, request_id=%s\n",
		payload.Username, payload.BountyID, payload.DeltaCompleted, payload.DeltaEarnings, payload.DeltaPosted, payload.DeltaCompletedAsEmployer, payload.RequestID)

	// 幂等性检查：是否已处理过该事件
	processed, err := c.store.IsProfileEventProcessed(context.Background(), payload.RequestID)
	if err != nil {
		log.Printf("检查事件是否已处理失败 [request_id=%s]: %v\n", payload.RequestID, err)
		msg.Nack(false, true) // 重试
		return
	}
	if processed {
		log.Printf("事件已处理过，跳过 [request_id=%s, username=%s]\n", payload.RequestID, payload.Username)
		msg.Ack(false)
		return
	}

	// 调用 store 刷新用户画像统计
	params := &models.RefreshStatsParams{
		BountyID:                 payload.BountyID,
		DeltaCompleted:           payload.DeltaCompleted,
		DeltaEarnings:           payload.DeltaEarnings,
		DeltaPosted:              payload.DeltaPosted,
		DeltaCompletedAsEmployer: payload.DeltaCompletedAsEmployer,
	}

	_, err = c.store.RefreshStats(context.Background(), payload.Username, params)
	if err != nil {
		log.Printf("刷新用户画像统计失败 [username=%s, bounty_id=%d]: %v\n", payload.Username, payload.BountyID, err)
		msg.Nack(false, true) // 重试
		return
	}

	// 记录已处理的事件
	if err := c.store.RecordProcessedProfileEvent(context.Background(), payload.RequestID, payload.Username, payload.BountyID); err != nil {
		log.Printf("记录已处理事件失败 [request_id=%s]: %v\n", payload.RequestID, err)
		// 不阻塞，消息已处理成功
	}

	log.Printf("用户画像统计更新成功：username=%s\n", payload.Username)
	msg.Ack(false)
}
