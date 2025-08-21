package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() (*gorm.DB, error) {
	dbHost, dbPort, dbUser, dbPass, dbName := os.Getenv("DATABASE_HOSTNAME"), os.Getenv("DATABASE_HOSTPORT"), os.Getenv("DATABASE_USERNAME"), os.Getenv("DATABASE_PASSWORD"), os.Getenv("DATABASE_NAME")
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPass == "" || dbName == "" {
		log.Fatalf("One or more datebase environment variables are not set. Please check your .env file")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)
	// fmt.Println(dsn)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{ // <--- 使用 mysql.Open
		Logger: logger.Default.LogMode(logger.Info), // 打印 SQL 日志
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接可复用的最长时间

	log.Println("Database connected successfully!")
	return db, nil
}

func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	for _, model := range models {
		err := db.AutoMigrate(model)
		if err != nil {
			return fmt.Errorf("failed to auto migrate table for model %T: %w", model, err)
		}
	}
	log.Println("Database migration completed successfully!")
	return nil
}
