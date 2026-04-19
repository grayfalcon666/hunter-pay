package models

import "time"

type Withdrawal struct {
	ID             int64     `gorm:"primaryKey;autoIncrement"`
	Username       string    `gorm:"type:varchar(255);not null;index"`
	AlipayRealName string    `json:"alipay_real_name"`
	AccountID      int64     `gorm:"not null"`
	Amount         int64     `gorm:"not null"`
	AlipayAccount  string    `gorm:"type:varchar(255);not null"`
	OutBizNo       string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	PayFundOrderID string    `gorm:"type:varchar(100);default:''"` // 支付宝返回的单号
	Status         string    `gorm:"type:varchar(50);not null;default:'PROCESSING';index"`
	ErrorMsg       string    `gorm:"type:text;default:''"` // 失败原因
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}
