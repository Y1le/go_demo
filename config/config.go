package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Email    EmailConfig
	OSS      OSSConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Retries  int
	Timeout  time.Duration
}

type OSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

type JWTConfig struct {
	Secret string
}

func LoadConfig() (*Config, error) {
	// 加载 .env 文件
	wd, _ := os.Getwd()
	log.Printf("Current working directory: %s", wd)

	envFile := ".env.local"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		envFile = ".env"
	}

	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			log.Printf("Warning: Error loading %s file: %v", envFile, err)
		} else {
			log.Printf("Loaded configuration from %s", envFile)
		}
	}

	// 手动构建配置结构
	cfg := &Config{
		Server: ServerConfig{
			Port: os.Getenv("APP_SERVER_PORT"),
		},
		Database: DatabaseConfig{
			DSN: os.Getenv("APP_DATABASE_DSN"),
		},
		Redis: RedisConfig{
			Addr:     os.Getenv("APP_REDIS_ADDR"),
			Password: os.Getenv("APP_REDIS_PASSWORD"),
			DB:       getEnvAsInt("APP_REDIS_DB", 0),
			PoolSize: getEnvAsInt("APP_REDIS_POOL_SIZE", 10),
		},
		Email: EmailConfig{
			Host:     os.Getenv("APP_EMAIL_HOST"),
			Port:     getEnvAsInt("APP_EMAIL_PORT", 465),
			Username: os.Getenv("APP_EMAIL_USERNAME"),
			Password: os.Getenv("APP_EMAIL_PASSWORD"),
			From:     os.Getenv("APP_EMAIL_FROM"),
			Retries:  getEnvAsInt("APP_EMAIL_RETRIES", 3),
			Timeout:  getEnvAsDuration("APP_EMAIL_TIMEOUT", 10*time.Second),
		},
		OSS: OSSConfig{
			Endpoint:        os.Getenv("APP_OSS_ENDPOINT"),
			AccessKeyID:     os.Getenv("APP_OSS_ACCESS_KEY_ID"),
			AccessKeySecret: os.Getenv("APP_OSS_ACCESS_KEY_SECRET"),
			BucketName:      os.Getenv("APP_OSS_BUCKET_NAME"),
		},
		JWT: JWTConfig{
			Secret: os.Getenv("APP_JWT_SECRET"),
		},
	}

	// 验证必需的配置
	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("APP_DATABASE_DSN is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("APP_JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	if val, err := time.ParseDuration(valStr); err == nil {
		return val
	}
	return defaultVal
}
