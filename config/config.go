package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Email    EmailConfig
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

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("./../yaml")
	viper.AddConfigPath("./cocnfig")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode into struct, %s", err)
		return nil, err
	}
	return &cfg, nil
}
