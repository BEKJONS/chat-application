package entity

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
}

type BlockedUser struct {
	gorm.Model
	UserID    uint // Who blocked
	BlockedID uint // Who is blocked
}
