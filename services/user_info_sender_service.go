package services

import (
	"context"
	"encoding/json"
	"fmt"
	"liam/repositories"
	"time"

	"github.com/segmentio/kafka-go"
)

// UserInfoMessage 定义要发送到 Kafka 的用户消息结构
type UserInfoMessage struct {
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	// 可以添加更多需要发送的用户信息
}

// UserInfoSenderService 定义发送用户信息的接口
type UserInfoSenderService interface {
	SendDailyUserInfo(ctx context.Context) error
}

type userInfoSenderServiceImpl struct {
	userRepo    repositories.UserRepository
	kafkaWriter *kafka.Writer // Kafka 生产者
	kafkaTopic  string        // Kafka Topic 名称
}

// NewUserInfoSenderService 创建一个新的 UserInfoSenderService 实例
func NewUserInfoSenderService(userRepo repositories.UserRepository, kafkaWriter *kafka.Writer, kafkaTopic string) UserInfoSenderService {
	return &userInfoSenderServiceImpl{
		userRepo:    userRepo,
		kafkaWriter: kafkaWriter,
		kafkaTopic:  kafkaTopic,
	}
}

// SendDailyUserInfo 每天发送用户信息的具体实现，将消息发送到 Kafka
func (s *userInfoSenderServiceImpl) SendDailyUserInfo(ctx context.Context) error {
	fmt.Println("Starting daily user info sending task...")

	// 1. 获取所有用户
	users, err := s.userRepo.GetAllUsers(ctx) // 假设您有一个 GetAllUsers 方法
	if err != nil {
		return fmt.Errorf("failed to get all users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found to send daily info.")
		return nil
	}

	// 2. 遍历用户，构建消息并发送到 Kafka
	messages := make([]kafka.Message, 0, len(users))
	for _, user := range users {
		// 确保用户有邮箱
		if user.Email == "" {
			fmt.Printf("Skipping user %d: no email address.\n", user.ID)
			continue
		}

		msgPayload := UserInfoMessage{
			UserID:    user.ID,
			Email:     user.Email,
			Username:  user.Name,
			CreatedAt: user.CreatedAt,
		}

		value, err := json.Marshal(msgPayload)
		if err != nil {
			fmt.Printf("Failed to marshal user %d info to JSON: %v\n", user.ID, err)
			continue
		}

		messages = append(messages, kafka.Message{
			Topic: s.kafkaTopic,                       // 指定 Topic
			Key:   []byte(fmt.Sprintf("%d", user.ID)), // 可以使用用户ID作为Key
			Value: value,
		})
	}

	if len(messages) > 0 {
		// 批量发送消息到 Kafka
		err = s.kafkaWriter.WriteMessages(ctx, messages...)
		if err != nil {
			return fmt.Errorf("failed to write %d messages to Kafka topic %s: %w", len(messages), s.kafkaTopic, err)
		}
		fmt.Printf("Successfully sent %d user info messages to Kafka topic %s.\n", len(messages), s.kafkaTopic)
	} else {
		fmt.Println("No valid user info messages to send to Kafka.")
	}

	fmt.Println("Daily user info sending task completed.")
	return nil
}
