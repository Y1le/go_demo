package services

import (
	"context"
	"fmt"
	"liam/repositories"
	"liam/utils"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	EmailVerificationCodePrefix = "email_code:"
	EmailVerificationCodeTTL    = 5 * time.Minute // 验证码有效期5分钟
)

type EmailService interface {
	SendVerificationEmail(ctx context.Context, email string) error
	VerifyEmailCode(ctx context.Context, email, code string) (bool, error)
}

type emailServiceImpl struct {
	emailSender utils.EmailSender
	redisRepo   repositories.RedisRepository
	userRepo    repositories.UserRepository // 用于检查邮箱是否存在
}

func NewEmailService(sender utils.EmailSender, redisRepo repositories.RedisRepository, userRepo repositories.UserRepository) EmailService {
	return &emailServiceImpl{
		emailSender: sender,
		redisRepo:   redisRepo,
		userRepo:    userRepo,
	}
}

func (s *emailServiceImpl) SendVerificationEmail(ctx context.Context, email string) error {
	// 检查邮箱是否已注册（可选，取决于业务逻辑）
	_, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil {
		// 如果用户已存在，可以返回错误，或者直接发送验证码（例如用于重置密码）
		return err
	}

	code := utils.GenerateVerificationCode()
	subject := "您的注册验证码"
	body := fmt.Sprintf(`
        <p>您好！</p>
        <p>您的注册验证码是：<strong>%s</strong></p>
        <p>请在 %d 分钟内使用此验证码完成注册。</p>
        <p>如果您没有请求此验证码，请忽略此邮件。</p>
    `, code, EmailVerificationCodeTTL/time.Minute)

	// 将验证码存储到 Redis，并设置过期时间
	key := EmailVerificationCodePrefix + email
	if err := s.redisRepo.Set(ctx, key, code, EmailVerificationCodeTTL); err != nil {
		return fmt.Errorf("failed to save verification code to redis: %w", err)
	}

	// 发送邮件
	if err := s.emailSender.SendEmail(email, subject, body); err != nil {
		return err
	}
	return nil
}

// VerifyEmailCode 验证邮箱验证码
func (s *emailServiceImpl) VerifyEmailCode(ctx context.Context, email, code string) (bool, error) {
	key := EmailVerificationCodePrefix + email
	storedCode, err := s.redisRepo.Get(ctx, key)
	if err != nil {
		if err == redis.Nil { // 如果键不存在，说明验证码已过期或未发送
			return false, fmt.Errorf("verification code for %s not found or expired", email)
		}
		return false, fmt.Errorf("failed to get verification code from redis: %w", err)
	}

	if storedCode == code {
		// 验证成功后，删除 Redis 中的验证码，防止重复使用
		_ = s.redisRepo.Del(ctx, key)
		return true, nil
	}
	return false, nil
}
