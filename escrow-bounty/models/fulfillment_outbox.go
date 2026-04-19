package models

import "time"

// FulfillmentOutbox 存储待发送的履约指数重算事件
type FulfillmentOutbox struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username   string    `gorm:"type:varchar(255);not null" json:"username"`
	Role       string    `gorm:"type:varchar(20);not null" json:"role"`
	BountyID   int64     `gorm:"not null" json:"bounty_id"`
	Status     string    `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	RetryCount int32     `gorm:"default:0" json:"retry_count"`
	MaxRetries int32     `gorm:"default:5" json:"max_retries"`
	LastError  string    `gorm:"type:text" json:"last_error,omitempty"`
	RequestID  string    `gorm:"type:varchar(255);not null" json:"request_id"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

const (
	FulfillmentOutboxStatusPending   = "PENDING"
	FulfillmentOutboxStatusCompleted = "COMPLETED"
	FulfillmentOutboxStatusFailed   = "FAILED"
)

func (FulfillmentOutbox) TableName() string {
	return "fulfillment_outbox"
}
