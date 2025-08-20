package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"` //主键
	Name      string    `json:"name" gorm:"size:255; not null; unique"`
	Email     string    `json:"email" gorm:"type:varchar(100); unique"`
	Age       int       `json:"age" gorm:"default: 18"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt
}
