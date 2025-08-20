package config

import (
	"fmt"
	"liam/models"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	dbHost, dbPort, dbUser, dbPass, dbName := os.Getenv("DATABASE_HOSTNAME"), os.Getenv("DATABASE_HOSTPORT"), os.Getenv("DATABASE_USERNAME"), os.Getenv("DATABASE_PASSWORD"), os.Getenv("DATABASE_NAME")
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPass == "" || dbName == "" {
		log.Fatalf("One or more datebase environment variables are not set. Please check your .env file")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)
	// fmt.Println(dsn)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Database connected successfully!")

	err = DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Failed to auto migrate: %v", err)
	}
}
