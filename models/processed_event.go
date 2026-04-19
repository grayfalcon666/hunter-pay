package models

import "time"

// ProcessedProfileEvent 用于记录已处理的 profile 更新事件，实现幂等性
type ProcessedProfileEvent struct {
	RequestID  string    `gorm:"type:varchar(255);primaryKey"`
	Username   string    `gorm:"type:varchar(255);not null;index"`
	BountyID   int64     `gorm:"not null"`
	ProcessedAt time.Time `gorm:"autoCreateTime"`
}
