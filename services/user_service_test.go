package services

import (
	"context"
	"liam/dto"
	"liam/models"
	"liam/pkg/errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

// CreateUser 模拟方法
func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// GetUserByID 模拟方法
func (m *MockUserRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UpdateUser 模拟方法
func (m *MockUserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// DeleteUser 模拟方法
func (m *MockUserRepository) DeleteUser(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetAllUsers 模拟方法
func (m *MockUserRepository) GetAllUser(ctx context.Context, offset, limit int) ([]models.User, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

// TestCreateUser_Success 测试创建用户成功
func TestUserService_CreateUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	req := &dto.CreateUserRequest{
		Name:  "Test User",
		Email: "test@example.com",
		Age:   25,
	}

	mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).
		Return(nil).
		Run(func(args mock.Arguments) {
			user := args.Get(1).(*models.User)
			user.ID = 1
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
		}).
		Once()
	resp, err := userService.CreateUser(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Name, resp.Name)
	assert.Equal(t, req.Email, resp.Email)
	assert.Equal(t, req.Age, resp.Age)
	assert.NotZero(t, resp.ID)
	mockRepo.AssertExpectations(t)
}

// TestCreateUser_Conflict 测试创建用户时发生冲突
func TestUserService_CreateUser_Conflict(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	req := &dto.CreateUserRequest{
		Name:  "Test User",
		Email: "existing@example.com",
		Age:   30,
	}

	// 预期 mockRepo.CreateUser 会被调用一次，并返回一个冲突错误
	conflictErr := errors.NewAppError(errors.ErrConflict.Code, "User with this email already exists", nil)
	mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(conflictErr).Once()

	resp, err := userService.CreateUser(context.Background(), req)

	assert.Error(t, err)                                          // 断言有错误
	assert.Nil(t, resp)                                           // 断言没有响应
	assert.True(t, errors.IsCustomError(err, errors.ErrConflict)) // 断言是自定义的冲突错误
	mockRepo.AssertExpectations(t)
}
