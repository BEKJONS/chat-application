package entity

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	SenderID      uint       `json:"sender_id"`
	ReceiverID    uint       `json:"receiver_id"` // For direct messages; 0 if group message
	GroupID       uint       `json:"group_id"`    // 0 if not a group message
	Content       string     `json:"content"`
	ScheduledTime *time.Time `json:"scheduled_time"` // Nil if sent immediately
	Sent          bool       `json:"sent" gorm:"default:false"`
}
