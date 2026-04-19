package models

import (
	"time"
)

type CreateReviewParams struct {
	ReviewedUsername string     `json:"reviewed_username"`
	BountyID         int64      `json:"bounty_id"`
	Rating           int        `json:"rating"`
	Comment          string     `json:"comment"`
	ReviewType       ReviewType `json:"review_type"`
}

// UserReview GORM model
type UserReview struct {
	ID               int64      `gorm:"primaryKey;autoIncrement"`
	ReviewerUsername string     `gorm:"type:varchar(255);not null;index"`
	ReviewedUsername string     `gorm:"type:varchar(255);not null;index:idx_reviews_reviewed"`
	BountyID         int64      `gorm:"not null;index:idx_reviews_bounty"`
	Rating           int        `gorm:"not null"`
	Comment          string     `gorm:"type:text;default:''"`
	ReviewType       ReviewType `gorm:"type:varchar(50);not null"`
	CreatedAt        time.Time  `gorm:"autoCreateTime"`
}
