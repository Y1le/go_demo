package services

import (
	"context"
	"liam/internal/dto"
	"liam/internal/models"
	"liam/pkg/errors"
	"liam/repositories"
	"liam/utils"
)

type UserService interface {
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, email, password string) (*dto.UserResponse, error)
	RegisterUser(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error)
	VerifyUserEmail(ctx context.Context, email, code string) error
	GetAllUser(ctx context.Context, pagination *dto.PaginationParams) ([]dto.UserResponse, int64, error)
	GetUserByID(ctx context.Context, id uint) (*dto.UserResponse, error)
	UpdateUser(ctx context.Context, id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	DeleteUser(ctx context.Context, id uint) error
}

type userServiceImpl struct {
	userRepo     repositories.UserRepository
	emailService EmailService
}

func NewUserService(userRepo repositories.UserRepository, emailService EmailService) UserService {
	return &userServiceImpl{userRepo: userRepo, emailService: emailService}
}

func (s *userServiceImpl) Login(ctx context.Context, email, password string) (*dto.UserResponse, error) {
	// 1. 查找用户
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err // 用户不存在
	}
	// 2. 验证密码
	if user.Password != password {
		return nil, errors.NewAppError(errors.ErrUnauthorized.Code, "Invalid password", nil)
	}

	token, err := utils.GenerateToken(user.ID, user.Name)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalError.Code, "Failed to generate token", err)
	}

	// 3. 返回用户响应
	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Age:       user.Age,
		Token:     token,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil

}

func (s *userServiceImpl) RegisterUser(ctx context.Context, req *dto.RegisterRequest) (*dto.UserResponse, error) {

	_, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, err
	}

	user := &models.User{
		Name:       req.Name,
		Email:      req.Email,
		Age:        req.Age,
		IsVerified: false,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// 3. 发送验证码邮件
	if err := s.emailService.SendVerificationEmail(ctx, req.Email); err != nil {
		_ = s.userRepo.DeleteUser(ctx, user.ID) // 回滚用户创建
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Age:       user.Age,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// VerifyUserEmail 验证用户邮箱
func (s *userServiceImpl) VerifyUserEmail(ctx context.Context, email, code string) error {
	// 1. 验证验证码
	isValid, err := s.emailService.VerifyEmailCode(ctx, email, code)
	if err != nil {
		return err // 验证码错误或过期
	}
	if !isValid {
		return errors.NewAppError(errors.ErrUnauthorized.Code, "Invalid verification code", nil)
	}

	// 2. 查找用户并更新验证状态
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return errors.NewAppError(errors.ErrConflict.Code, "需要验证邮箱未注册", err) // 用户不存在
	}

	if user.IsVerified {
		// return &repositories.AppError{Code: repositories.ErrorCodeConflict, Message: "Email already verified"}
		return errors.NewAppError(errors.ErrConflict.Code, "邮箱已注册", nil)
	}

	user.IsVerified = true
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return errors.NewAppError(errors.ErrInternalError.Code, "Failed to update user verification status", err)
	}
	return nil
}

func (s *userServiceImpl) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	user := &models.User{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err // 直接返回 Repository 层的 AppError
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Age:       user.Age,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *userServiceImpl) GetUserByID(ctx context.Context, id uint) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Age:       user.Age,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *userServiceImpl) UpdateUser(ctx context.Context, id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err // 用户不存在
	}

	// 根据 DTO 更新模型字段
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Age != nil {
		user.Age = *req.Age
	}

	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Age:       user.Age,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *userServiceImpl) DeleteUser(ctx context.Context, id uint) error {
	return s.userRepo.DeleteUser(ctx, id)
}

func (s *userServiceImpl) GetAllUser(ctx context.Context, pagination *dto.PaginationParams) ([]dto.UserResponse, int64, error) {
	offset := (pagination.Page - 1) * pagination.PageSize
	limit := pagination.PageSize

	users, total, err := s.userRepo.GetAllUser(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	var userResponses []dto.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, dto.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Age:       user.Age,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return userResponses, total, nil
}

// func (s *userServiceImpl) CreateUser(user *models.User) error {
// 	return s.db.Create(user).Error
// }

// func (s *userServiceImpl) GetAllUser() ([]models.User, error) {
// 	var users []models.User
// 	err := s.db.Find(&users).Error
// 	return users, err
// }

// func (s *userServiceImpl) GetUserByID(id uint) (*models.User, error) {
// 	var user models.User
// 	err := s.db.First(&user, id).Error
// 	if err == gorm.ErrRecordNotFound {
// 		return nil, nil
// 	}
// 	return &user, err
// }

// func (s *userServiceImpl) UpdateUser(user *models.User) error {
// 	return s.db.Save(user).Error
// }

// func (s *userServiceImpl) DeleteUser(id uint) error {
// 	res := s.db.Delete(&models.User{}, id)
// 	if res.Error != nil {
// 		return res.Error
// 	}
// 	if res.RowsAffected == 0 {
// 		return gorm.ErrRecordNotFound
// 	}
// 	return nil
// }
