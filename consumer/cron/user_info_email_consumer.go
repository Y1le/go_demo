package main

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/joho/godotenv"
// 	"github.com/segmentio/kafka-go"

// 	"liam/config"
// 	"liam/services"
// 	"liam/utils"
// )

// func main() {
// 	// 1. 加载配置
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Fatalf("Error loading .env file: %v", err)
// 	}

// 	cfg, err := config.LoadConfig()
// 	if err != nil {
// 		log.Fatalf("Failed to load config: %v", err)
// 	}

// 	// 配置日志
// 	logWriter, err := utils.NewDailyLogWriter("logs", "email_consumer") // 消费者使用单独的日志文件
// 	if err != nil {
// 		log.Fatalf("Failed to create daily log writer: %v", err)
// 	}
// 	defer logWriter.Close()
// 	multiWriter := io.MultiWriter(os.Stdout, logWriter)
// 	log.SetOutput(multiWriter)                   // 设置日志输出到文件和控制台
// 	log.SetFlags(log.LstdFlags | log.Lshortfile) // 添加文件名和行号

// 	log.Println("Starting Kafka email consumer...")

// 	// 2. 初始化邮件发送器 (消费者需要它来发送邮件)
// 	emailSender := utils.NewEmailSender(&cfg.Email)

// 	// 3. 初始化 Kafka Reader (消费者)
// 	r := kafka.NewReader(kafka.ReaderConfig{
// 		Brokers:        []string{cfg.Kafka.Broker},
// 		Topic:          cfg.Kafka.Topic,
// 		GroupID:        "daily-user-info-email-group", // 消费者组ID，确保唯一性
// 		MinBytes:       10e3,                          // 10KB
// 		MaxBytes:       10e6,                          // 10MB
// 		MaxWait:        1 * time.Second,               // 最长等待时间
// 		CommitInterval: 1 * time.Second,               // 自动提交偏移量
// 		Logger:         kafka.LoggerFunc(log.Printf),  // 添加 Kafka 日志
// 		ErrorLogger:    kafka.LoggerFunc(log.Printf),
// 	})
// 	defer func() {
// 		if err := r.Close(); err != nil {
// 			log.Printf("Failed to close Kafka reader: %v", err)
// 		} else {
// 			log.Println("Kafka reader closed.")
// 		}
// 	}()

// 	log.Printf("Kafka consumer started for topic: %s, group: %s", cfg.Kafka.Topic, "daily-user-info-email-group")

// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// 	for {
// 		select {
// 		case <-sigChan:
// 			return
// 		default:
// 			// 从 Kafka 拉取消息
// 			ctxRead, cancelRead := context.WithTimeout(context.Background(), 15*time.Second) // 读取消息的超时时间
// 			message, err := r.FetchMessage(ctxRead)
// 			cancelRead() // 及时取消 context

// 			if err != nil {
// 				if err == context.DeadlineExceeded {
// 					// 正常超时，没有新消息，继续循环
// 					continue
// 				}
// 				log.Printf("Error fetching message from Kafka: %v", err)
// 				time.Sleep(time.Second) // 稍作等待，避免CPU空转
// 				continue
// 			}

// 			var userInfo services.UserInfoMessage // 使用 services 包中定义的结构体
// 			err = json.Unmarshal(message.Value, &userInfo)
// 			if err != nil {
// 				log.Printf("Error unmarshalling message value: %v, message: %s", err, string(message.Value))
// 				// 记录错误，并提交此消息的偏移量，避免重复处理错误消息
// 				if commitErr := r.CommitMessages(context.Background(), message); commitErr != nil {
// 					log.Printf("Failed to commit message after unmarshalling error: %v", commitErr)
// 				}
// 				continue
// 			}

// 			log.Printf("Received message for user: %s (ID: %d, Email: %s)", userInfo.Username, userInfo.UserID, userInfo.Email)

// 			// 构造邮件内容
// 			subject := "您的每日用户信息"
// 			body := fmt.Sprintf(`
//                 <p>您好，%s！</p>
//                 <p>这是您的每日用户信息：</p>
//                 <ul>
//                     <li>用户ID: %d</li>
//                     <li>用户名: %s</li>
//                     <li>邮箱: %s</li>
//                     <li>创建时间: %s</li>
//                 </ul>
//                 <p>祝您有美好的一天！</p>
//             `, userInfo.Username, userInfo.UserID, userInfo.Username, userInfo.Email, userInfo.CreatedAt.Format(time.RFC1123))

// 			// 发送邮件
// 			_, sendCancel := context.WithTimeout(context.Background(), time.Duration(cfg.Email.Timeout)) // 使用配置的邮件超时时间
// 			sendErr := emailSender.SendEmail(userInfo.Email, subject, body)
// 			sendCancel()

// 			if sendErr != nil {
// 				log.Printf("Failed to send daily user info email to %s (User ID: %d): %v", userInfo.Email, userInfo.UserID, sendErr)
// 				// 邮件发送失败，不提交此消息的偏移量，以便在下次拉取时重试
// 				// 在实际生产中，可以考虑将失败消息发送到死信队列 (DLQ)
// 			} else {
// 				log.Printf("Successfully sent daily user info email to %s (User ID: %d)", userInfo.Email, userInfo.UserID)
// 				// 邮件发送成功，提交消息偏移量
// 				if commitErr := r.CommitMessages(context.Background(), message); commitErr != nil {
// 					log.Printf("Failed to commit message after successful email send: %v", commitErr)
// 				}
// 			}
// 		}
// 	}
// }
