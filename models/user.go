package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Name  string `json:"name" gorm:"size:255; not null; unique"`
	Email string `json:"email" gorm:"type:varchar(100); unique"`
	Age   int    `json:"age" gorm:"default: 18"`

	Password   string `json:"password" gorm:"default: 123456"`
	IsVerified bool   `gorm:"default:false"`
}
