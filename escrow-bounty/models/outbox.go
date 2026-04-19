package models

import "time"

// OutboxStatus represents the processing status of an outbox entry
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusCompleted OutboxStatus = "COMPLETED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

// ProfileUpdateOutbox 存储待发送的用户画像更新事件
type ProfileUpdateOutbox struct {
	ID                         int64        `gorm:"primaryKey;autoIncrement" json:"id"`
	Username                   string       `gorm:"type:varchar(255);not null;index" json:"username"`
	BountyID                   int64        `gorm:"not null" json:"bounty_id"`
	DeltaCompleted             int32        `gorm:"default:0" json:"delta_completed"`
	DeltaEarnings              int64        `gorm:"default:0" json:"delta_earnings"`
	DeltaPosted                int32        `gorm:"default:0" json:"delta_posted"`
	DeltaCompletedAsEmployer   int32        `gorm:"default:0" json:"delta_completed_as_employer"`
	Status                     OutboxStatus `gorm:"type:varchar(20);not null;default:'PENDING';index" json:"status"`
	RetryCount                 int32        `gorm:"default:0" json:"retry_count"`
	MaxRetries                 int32        `gorm:"default:5" json:"max_retries"`
	LastError                  string       `gorm:"type:text" json:"last_error,omitempty"`
	RequestID                  string       `gorm:"type:varchar(255);not null;uniqueIndex" json:"request_id"`
	CreatedAt                  time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                  time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ProfileUpdateOutbox) TableName() string {
	return "profile_update_outbox"
}
