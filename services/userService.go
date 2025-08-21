package services

import (
	"context"
	"liam/dto"
	"liam/models"
	"liam/repositories"
)

type UserService interface {
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	GetAllUser(ctx context.Context, pagination *dto.PaginationParams) ([]dto.UserResponse, int64, error)
	GetUserByID(ctx context.Context, id uint) (*dto.UserResponse, error)
	UpdateUser(ctx context.Context, id uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	DeleteUser(ctx context.Context, id uint) error
}

type userServiceImpl struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userServiceImpl{userRepo: userRepo}
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
