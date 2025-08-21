package services

import (
	"context"
	"liam/config"
	"liam/dto"
	"liam/models"
	"liam/pkg/errors"
	"liam/repositories"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var (
	testDB      *gorm.DB
	userService UserService
)

func TestMain(m *testing.M) {
	// 获取当前测试文件的目录 (services 目录)
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)

	// 构建到项目根目录的 .env 文件的路径
	// 假设 .env 文件在项目根目录，即 services 目录的上一级
	projectRoot := filepath.Join(currentDir, "..") // 上一级目录
	envPath := filepath.Join(projectRoot, ".env")

	// 尝试加载 .env 文件
	err := godotenv.Load(envPath) // <--- 指定 .env 文件的完整路径
	if err != nil {
		// 打印更详细的错误信息，包括尝试加载的路径
		// 注意：如果 .env 文件是可选的，这里可以改为 log.Printf 而不是 log.Fatalf
		// 但对于数据库凭据，通常是必须的，所以 Fatalf 更合适
		log.Fatalf("Error loading .env file from %s: %v", envPath, err)
	}
	// 1. 初始化内存数据库
	testDB, err = config.InitDB()
	if err != nil {
		panic("Failed to  connect to test database: " + err.Error())
	}

	err = config.AutoMigrate(testDB, &models.User{})
	if err != nil {
		panic("Failed to auto migrate test database: " + err.Error())
	}

	userRepo := repositories.NewUserRepository(testDB)
	userService = NewUserService(userRepo)

	code := m.Run()

	sqlDB, _ := testDB.DB()
	sqlDB.Close()

	os.Exit(code)
}

func setupTest(t *testing.T) {
	err := testDB.Exec("DELETE FROM users").Error
	assert.NoError(t, err)
	// testDB.Exec("DELETE FROM sqlite_sequence WHERE name='users'") // <--- 移除或注释掉这行
}

func TestUserService_Integration_CreateAndGet(t *testing.T) {
	setupTest(t)

	req := &dto.CreateUserRequest{
		Name:  "Test User Integration",
		Email: "Test Email Integration",
		Age:   30,
	}

	ctx := context.Background()
	createdUser, err := userService.CreateUser(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.NotZero(t, createdUser.ID)
	assert.Equal(t, req.Email, createdUser.Email)

	fetchedUser, err := userService.GetUserByID(ctx, createdUser.ID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedUser)
	assert.Equal(t, createdUser.ID, fetchedUser.ID)
	assert.Equal(t, createdUser.Name, fetchedUser.Name)
	assert.Equal(t, createdUser.Email, fetchedUser.Email)
}

func TestUserService_Integration_DuplicateEmail(t *testing.T) {
	setupTest(t)

	ctx := context.Background()
	req1 := &dto.CreateUserRequest{
		Name:  "user One",
		Email: "1@example.com",
		Age:   30,
	}
	_, err := userService.CreateUser(ctx, req1)
	assert.NoError(t, err)

	req2 := &dto.CreateUserRequest{
		Name:  "user Two",
		Email: "1@example.com",
		Age:   22,
	}
	_, err = userService.CreateUser(ctx, req2)
	assert.Error(t, err)
	// 并且预期是冲突错误

	assert.True(t, errors.IsCustomError(err, errors.ErrInternalError))

}
