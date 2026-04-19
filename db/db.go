package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Client *gorm.DB

func InitDB(dsn string) {
	var err error
	Client, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	sqlDB, err := Client.DB()
	if err != nil {
		log.Fatalf("无法获取底层 DB 连接: %v", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
}
