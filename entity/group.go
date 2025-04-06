package entity

import "gorm.io/gorm"

type Group struct {
	gorm.Model
	Name    string
	Members []GroupMember
}

type GroupMember struct {
	gorm.Model
	GroupID uint
	UserID  uint
}
