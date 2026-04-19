package models

import "time"

type Payment struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	Username      string    `gorm:"type:varchar(255);not null;index"`
	AccountID     int64     `gorm:"not null"`
	Amount        int64     `gorm:"not null"`
	Currency      string    `gorm:"type:varchar(10);not null;default:'CNY'"`
	OutTradeNo    string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	AlipayTradeNo string    `gorm:"type:varchar(100);default:''"`
	Status        string    `gorm:"type:varchar(50);not null;default:'PENDING';index"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}
