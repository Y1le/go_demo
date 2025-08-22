package utils

import (
	"context"
	"fmt"
	"liam/config"
	"log"
	"time"

	"gopkg.in/gomail.v2"
)

type EmailSender interface {
	SendEmail(to, subject, bod string) error
}

type emailSenderImpl struct {
	dialer  *gomail.Dialer
	from    string
	retries int
	timeout time.Duration
}

func NewEmailSender(cfg *config.EmailConfig) EmailSender {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	return &emailSenderImpl{
		dialer:  d,
		from:    cfg.From,
		retries: cfg.Retries,
		timeout: cfg.Timeout,
	}
}

func (s *emailSenderImpl) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	log.Println(body)
	return nil
	var err error
	for i := 0; i < s.retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			defer cancel()
			done <- s.dialer.DialAndSend(m)
		}()

		select {
		case <-ctx.Done():
			err := fmt.Errorf("email sending timed out after %s", s.timeout)
			log.Printf("Attempt %d: %v", i+1, err)
		case sendErr := <-done:
			if sendErr == nil {
				return nil // 邮件发送成功
			}
			log.Printf("Attempt %d: Failed to send email to %s. Error: %v", i+1, to, sendErr)
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Failed to send email to %s after %d retries: %w", to, s.retries, err)
}

func GenerateVerificationCode() string {
	return fmt.Sprintf("%06d", time.Now().Nanosecond()*13331%1000000)
}
