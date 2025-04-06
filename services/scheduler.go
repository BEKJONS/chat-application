package services

import (
	"chat_app/entity"
	"log"
	"time"

	"gorm.io/gorm"
)

type SchedulerService struct {
	DB               *gorm.DB
	WebSocketService *WebSocketService
	Ticker           *time.Ticker
	Done             chan bool
}

func NewSchedulerService(db *gorm.DB, wsService *WebSocketService) *SchedulerService {
	ss := &SchedulerService{
		DB:               db,
		WebSocketService: wsService,
		Ticker:           time.NewTicker(10 * time.Second), // Check every 10 seconds
		Done:             make(chan bool),
	}
	go ss.run()
	return ss
}

func (ss *SchedulerService) run() {
	for {
		select {
		case <-ss.Done:
			ss.Ticker.Stop()
			return
		case t := <-ss.Ticker.C:
			log.Printf("Checking for scheduled messages at %v", t)
			ss.processScheduledMessages()
		}
	}
}

func (ss *SchedulerService) processScheduledMessages() {
	var messages []entity.Message
	now := time.Now().UTC()
	windowStart := now.Add(-1 * time.Second) // 1 second before now
	windowEnd := now.Add(59 * time.Second)   // 59 seconds after now

	log.Printf("Current time (UTC): %v, Window: [%v, %v]", now, windowStart, windowEnd)

	// Find messages within the window that haven't been sent
	if err := ss.DB.Where("scheduled_time >= ? AND scheduled_time <= ? AND sent = ?", windowStart, windowEnd, false).Find(&messages).Error; err != nil {
		log.Printf("Error fetching scheduled messages: %v", err)
		return
	}

	for _, msg := range messages {
		log.Printf("Processing scheduled message ID %d, scheduled for %v", msg.ID, msg.ScheduledTime)

		// Send the message via the WebSocketService's Broadcast channel
		ss.WebSocketService.Broadcast <- msg

		// Mark the message as sent
		msg.Sent = true
		if err := ss.DB.Save(&msg).Error; err != nil {
			log.Printf("Error marking message ID %d as sent: %v", msg.ID, err)
		}
	}
}

func (ss *SchedulerService) Stop() {
	ss.Done <- true
}
