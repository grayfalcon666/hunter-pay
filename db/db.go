package db

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Client *gorm.DB

func InitDB(dsn string) {
	var err error

	// 配置 GORM：在开发阶段开启 Info 级别的日志 这样终端会打印出 GORM 实际执行的 SQL 语句 利于调试
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	Client, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		log.Fatalf("cannot connect to the DB: %v", err)
	}

	sqlDB, err := Client.DB()
	if err != nil {
		log.Fatalf("cannot get the *sql.db: %v", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)           // 空闲时的最大连接数
	sqlDB.SetMaxOpenConns(100)          // 数据库的最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接可复用的最大时间

	log.Println("数据库连接成功并已初始化连接池！")
}
