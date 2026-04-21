package mq

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

const (
	// 主业务配置
	PaymentExchange = "payment_exchange"

	// 死信业务配置 (Dead Letter Exchange)
	PaymentDLX = "payment_dlx"
	PaymentDLQ = "payment_dlq"
)

type Producer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	amqpURL  string
	mu       sync.Mutex
	confirms chan amqp.Confirmation
}

func NewProducer(amqpURL string) (*Producer, error) {
	p := &Producer{amqpURL: amqpURL}
	if err := p.connect(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Producer) connect() error {
	conn, err := amqp.Dial(p.amqpURL)
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("打开 Channel 失败: %w", err)
	}

	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("开启 Publisher Confirms 失败: %w", err)
	}

	p.confirms = make(chan amqp.Confirmation, 100)
	ch.NotifyPublish(p.confirms)

	// 声明死信交换机
	err = ch.ExchangeDeclare(PaymentDLX, "direct", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明 DLX 失败: %w", err)
	}

	// 声明死信队列
	_, err = ch.QueueDeclare(PaymentDLQ, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明 DLQ 失败: %w", err)
	}

	// 绑定死信队列到死信交换机
	err = ch.QueueBind(PaymentDLQ, PaymentRoutingKey, PaymentDLX, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("绑定 DLQ 失败: %w", err)
	}

	// 声明主交换机
	err = ch.ExchangeDeclare(PaymentExchange, "direct", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明主 Exchange 失败: %w", err)
	}

	// 声明主队列
	args := amqp.Table{
		"x-dead-letter-exchange": PaymentDLX,
	}
	_, err = ch.QueueDeclare(PaymentQueue, true, false, false, false, args)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明支付 Queue 失败: %w", err)
	}
	err = ch.QueueBind(PaymentQueue, PaymentRoutingKey, PaymentExchange, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("绑定支付 Queue 失败: %w", err)
	}

	// 声明提现队列
	_, err = ch.QueueDeclare(WithdrawalQueue, true, false, false, false, args)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明提现 Queue 失败: %w", err)
	}
	err = ch.QueueBind(WithdrawalQueue, WithdrawalRoutingKey, PaymentExchange, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("绑定提现 Queue 失败: %w", err)
	}

	p.conn = conn
	p.channel = ch
	return nil
}

// reconnect 重建 channel，用于超时或错误后恢复
func (p *Producer) reconnect() error {
	// channel.Close() 会同步等待 RabbitMQ 确认帧，如果 RabbitMQ 已单方面关闭了
	// channel（比如 Confirm 超时），Close() 会永远卡住。因此用 goroutine + 超时强制关闭。
	if p.channel != nil {
		done := make(chan struct{})
		go func() {
			p.channel.Close()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			log.Println("channel.Close() 超时，强制关闭")
		}
	}
	if p.conn != nil {
		done := make(chan struct{})
		go func() {
			p.conn.Close()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			log.Println("conn.Close() 超时，强制继续")
		}
	}
	log.Println("RabbitMQ Producer 正在重建连接...")
	return p.connect() // 这里的 connect() 会重新初始化 p.confirms
}

func (p *Producer) publish(body []byte, routingKey, bizNo string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.channel.Publish(
		PaymentExchange,
		routingKey,
		true,  // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	select {
	case confirmed := <-p.confirms:
		if confirmed.Ack {
			log.Printf("消息已安全送达 MQ 磁盘! 订单号: %s\n", bizNo)
			return nil
		}
		return fmt.Errorf("RabbitMQ 拒绝了该消息 (Nack)")
	case <-time.After(5 * time.Second):
		// 超时后重建 channel 再重试一次
		log.Printf("MQ Confirm 超时，重建 channel... 订单号: %s\n", bizNo)
		if reconnErr := p.reconnect(); reconnErr != nil {
			return fmt.Errorf("重建 channel 失败: %v，原始超时: 等待 Confirm 超时", reconnErr)
		}
		return fmt.Errorf("等待 RabbitMQ 确认超时，已重建 channel，请重试")
	}
}

func (p *Producer) PublishPaymentSuccess(msg *PaymentSuccessMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.publish(body, PaymentRoutingKey, msg.OutTradeNo)
}

func (p *Producer) PublishWithdrawalProcess(msg *WithdrawalMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.publish(body, WithdrawalRoutingKey, msg.OutBizNo)
}

// Close 优雅关闭连接
func (p *Producer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
