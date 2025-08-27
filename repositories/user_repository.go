package repositories

import (
	"context"
	"liam/models"
	"liam/pkg/errors"

	stdErr "errors"

	"gorm.io/gorm"
)

//Repository 层负责与数据库进行直接交互，它不包含任何业务逻辑，只提供 CRUD 操作

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetAllUser(ctx context.Context, offset, limit int) ([]models.User, int64, error)
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uint) error
	GetAllUsers(ctx context.Context) ([]models.User, error)
}

type userRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) CreateUser(ctx context.Context, user *models.User) error {
	// return s.db.Create(user).Error
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		if stdErr.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errors.NewAppError(errors.ErrConflict.Code, "User with this email or name already exists", result.Error)
		}
		return errors.NewAppError(errors.ErrInternalError.Code, "Failed to create user in database", result.Error)
	}
	return nil

}

func (r *userRepositoryImpl) GetAllUser(ctx context.Context, offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	countResult := r.db.WithContext(ctx).Model(&models.User{}).Count(&total)
	if countResult.Error != nil {
		return nil, 0, errors.NewAppError(errors.ErrInternalError.Code, "Failed to count users", countResult.Error)
	}

	result := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users)
	if result.Error != nil {
		return nil, 0, errors.NewAppError(errors.ErrInternalError.Code, "Failed to retrieve users from database", result.Error)
	}
	return users, total, nil
}

func (r *userRepositoryImpl) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).First(&user, id)
	if result.Error != nil {
		if stdErr.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.NewAppError(errors.ErrNotFound.Code, "User not found", result.Error)
		}
		return nil, errors.NewAppError(errors.ErrInternalError.Code, "Failed to retrieve user from database", result.Error)
	}
	return &user, nil
}

func (r *userRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if stdErr.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.NewAppError(errors.ErrNotFound.Code, "User not found with this email", result.Error)
		}
		return nil, errors.NewAppError(errors.ErrInternalError.Code, "Failed to get user by email", result.Error)
	}
	return &user, nil
}

func (r *userRepositoryImpl) UpdateUser(ctx context.Context, user *models.User) error {
	// GORM 的 Save 方法会根据主键更新所有字段，如果只想更新非零值字段，可以使用 Updates
	// result := r.db.WithContext(ctx).Model(user).Updates(user)
	result := r.db.WithContext(ctx).Save(user) // Save 会更新所有字段
	if result.Error != nil {
		return errors.NewAppError(errors.ErrInternalError.Code, "Failed to update user in database", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewAppError(errors.ErrNotFound.Code, "User not found for update", nil)
	}
	return nil
}

func (r *userRepositoryImpl) DeleteUser(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	if result.Error != nil {
		return errors.NewAppError(errors.ErrInternalError.Code, "Failed to delete user from database", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewAppError(errors.ErrNotFound.Code, "User not found or already deleted", nil)
	}
	return nil
}

func (r *userRepositoryImpl) GetAllUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
