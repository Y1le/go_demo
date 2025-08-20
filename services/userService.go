package services

import (
	"liam/models"

	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(user *models.User) error
	GetAllUser() ([]models.User, error)
	GetUserByID(id uint) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id uint) error
}

type userServiceImpl struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) UserService {
	return &userServiceImpl{db: db}
}

func (s *userServiceImpl) CreateUser(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *userServiceImpl) GetAllUser() ([]models.User, error) {
	var users []models.User
	err := s.db.Find(&users).Error
	return users, err
}

func (s *userServiceImpl) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &user, err
}

func (s *userServiceImpl) UpdateUser(user *models.User) error {
	return s.db.Save(user).Error
}

func (s *userServiceImpl) DeleteUser(id uint) error {
	res := s.db.Delete(&models.User{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
