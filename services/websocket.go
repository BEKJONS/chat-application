package services

import (
	"chat_app/entity"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing
	},
}

type Client struct {
	Conn   *websocket.Conn
	UserID uint
}

type WebSocketService struct {
	DB        *gorm.DB
	Clients   map[*Client]bool
	Mutex     sync.Mutex
	Broadcast chan entity.Message
}

func NewWebSocketService(db *gorm.DB) *WebSocketService {
	ws := &WebSocketService{
		DB:        db,
		Clients:   make(map[*Client]bool),
		Broadcast: make(chan entity.Message),
	}
	go ws.handleMessages()
	return ws
}

func (ws *WebSocketService) HandleConnections(w http.ResponseWriter, r *http.Request, userID uint) {
	log.Printf("WebSocket request headers: %v", r.Header)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	client := &Client{Conn: conn, UserID: userID}
	ws.Mutex.Lock()
	ws.Clients[client] = true
	ws.Mutex.Unlock()

	client.Conn.WriteJSON(map[string]interface{}{
		"message": "Connected to chat",
		"user_id": userID,
	})

	go ws.handleClient(client)
}

func (ws *WebSocketService) handleClient(client *Client) {
	defer func() {
		ws.Mutex.Lock()
		delete(ws.Clients, client)
		ws.Mutex.Unlock()
		client.Conn.Close()
		log.Printf("Client disconnected: user_id=%d", client.UserID)
	}()

	for {
		var msg entity.Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Deserialized message: %+v", msg)

		msg.SenderID = client.UserID
		log.Printf("Message after setting SenderID: %+v", msg)

		if msg.ScheduledTime != nil {
			// Validate ScheduledTime
			if msg.ScheduledTime.Before(time.Now().UTC()) {
				log.Printf("Scheduled time is in the past: %v", msg.ScheduledTime)
				client.Conn.WriteJSON(map[string]string{
					"error": "Scheduled time must be in the future",
				})
				continue
			}
			log.Printf("Saving scheduled message to DB: %+v", msg)
			if err := ws.DB.Create(&msg).Error; err != nil {
				log.Printf("Error saving scheduled message: %v", err)
				client.Conn.WriteJSON(map[string]string{
					"error": "Failed to schedule message",
				})
				continue
			}
			client.Conn.WriteJSON(map[string]interface{}{
				"message": "Message scheduled successfully",
				"id":      msg.ID,
			})
		} else {
			log.Printf("Sending message to Broadcast channel: %+v", msg)
			ws.Broadcast <- msg
		}
	}
}

func (ws *WebSocketService) handleMessages() {
	for msg := range ws.Broadcast {
		log.Printf("Processing message: %+v", msg)

		// For group messages, check membership before saving to the database
		if msg.GroupID != 0 {
			var senderMembership entity.GroupMember
			if err := ws.DB.Where("group_id = ? AND user_id = ? AND deleted_at IS NULL", msg.GroupID, msg.SenderID).First(&senderMembership).Error; err != nil {
				log.Printf("Sender (user_id=%d) is not a member of group %d or is soft-deleted, skipping message", msg.SenderID, msg.GroupID)
				ws.Mutex.Lock()
				for client := range ws.Clients {
					if client.UserID == msg.SenderID {
						errorMsg := map[string]string{
							"error": "You are not a member of this group or have been removed.",
						}
						if err := client.Conn.WriteJSON(errorMsg); err != nil {
							log.Printf("Error sending error message to user %d: %v", client.UserID, err)
						}
						break
					}
				}
				ws.Mutex.Unlock()
				continue
			}
		}

		// Save the message to the database (only if the sender is a member for group messages)
		if msg.ID == 0 { // Only save if the message hasn't been saved yet (e.g., for immediate messages)
			if err := ws.DB.Create(&msg).Error; err != nil {
				log.Printf("Error saving message to database: %v", err)
				continue
			}
			log.Printf("Message saved to database with ID: %d", msg.ID)
		}

		ws.Mutex.Lock()
		for client := range ws.Clients {
			log.Printf("Checking client: user_id=%d", client.UserID)

			// Direct message
			if msg.ReceiverID != 0 && client.UserID == msg.ReceiverID {
				var blocked entity.BlockedUser
				if err := ws.DB.Where("user_id = ? AND blocked_id = ?", msg.ReceiverID, msg.SenderID).First(&blocked).Error; err == nil {
					log.Printf("User %d has blocked user %d, skipping direct message", msg.ReceiverID, msg.SenderID)
					for senderClient := range ws.Clients {
						if senderClient.UserID == msg.SenderID {
							errorMsg := map[string]string{
								"error": "You have been blocked by the recipient.",
							}
							if err := senderClient.Conn.WriteJSON(errorMsg); err != nil {
								log.Printf("Error sending error message to user %d: %v", senderClient.UserID, err)
							}
							break
						}
					}
					continue
				}
				log.Printf("Sending direct message to user %d", client.UserID)
				client.Conn.WriteJSON(msg)
				continue
			}

			// Group message
			if msg.GroupID != 0 {
				var members []entity.GroupMember
				if err := ws.DB.Where("group_id = ? AND deleted_at IS NULL", msg.GroupID).Find(&members).Error; err != nil {
					log.Printf("Error fetching group members for group %d: %v", msg.GroupID, err)
					continue
				}
				log.Printf("Group %d members (excluding soft-deleted): %v", msg.GroupID, members)

				for _, member := range members {
					if client.UserID == member.UserID {
						var blocked entity.BlockedUser
						if err := ws.DB.Where("user_id = ? AND blocked_id = ?", client.UserID, msg.SenderID).First(&blocked).Error; err == nil {
							log.Printf("User %d has blocked user %d, skipping group message", client.UserID, msg.SenderID)
							continue
						}
						log.Printf("Sending group message to user %d", client.UserID)
						if err := client.Conn.WriteJSON(msg); err != nil {
							log.Printf("Error sending message to user %d: %v", client.UserID, err)
						}
						break
					}
				}
				continue
			}

			// Broadcast message
			if msg.ReceiverID == 0 && msg.GroupID == 0 {
				log.Printf("Sending broadcast message to user %d", client.UserID)
				client.Conn.WriteJSON(msg)
				continue
			}

			log.Printf("Message not sent to user %d: no matching condition (ReceiverID=%d, GroupID=%d)", client.UserID, msg.ReceiverID, msg.GroupID)
		}
		ws.Mutex.Unlock()

		// Mark the message as sent
		if !msg.Sent {
			msg.Sent = true
			if err := ws.DB.Save(&msg).Error; err != nil {
				log.Printf("Error marking message as sent: %v", err)
			}
		}
	}
}
