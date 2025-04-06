package handler

import (
	"chat_app/entity"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"net/http"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	log.Printf("Received register request: %v", r.Body)
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		log.Printf("Failed to decode body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if creds.Username == "" || creds.Password == "" {
		log.Printf("Empty username or password")
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	user := entity.User{
		Username: creds.Username,
		Password: string(hashedPassword),
	}
	if err := h.AuthService.DB.Create(&user).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"username": creds.Username,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	log.Printf("Received login request headers: %v", r.Header)
	log.Printf("Received login request body: %v", r.Body)
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		log.Printf("Failed to decode body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("Parsed credentials: username=%s, password=%s", creds.Username, creds.Password)
	if creds.Username == "" || creds.Password == "" {
		log.Printf("Empty username or password")
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	var user entity.User
	if err := h.AuthService.DB.Where("username = ?", creds.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"username": creds.Username,
	})
}

func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	var blockRequest struct {
		UserID    uint `json:"user_id"`    // The user who is blocking
		BlockedID uint `json:"blocked_id"` // The user to block
	}
	if err := json.NewDecoder(r.Body).Decode(&blockRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if blockRequest.UserID == 0 || blockRequest.BlockedID == 0 {
		http.Error(w, "user_id and blocked_id are required", http.StatusBadRequest)
		return
	}
	if blockRequest.UserID == blockRequest.BlockedID {
		http.Error(w, "Cannot block yourself", http.StatusBadRequest)
		return
	}

	// Check if the user exists
	var user entity.User
	if err := h.AuthService.DB.First(&user, blockRequest.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}

	// Check if the user to block exists
	var blockedUser entity.User
	if err := h.AuthService.DB.First(&blockedUser, blockRequest.BlockedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User to block not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding user to block", http.StatusInternalServerError)
		return
	}

	// Check if the user is already blocked
	var existingBlock entity.BlockedUser
	if err := h.AuthService.DB.Where("user_id = ? AND blocked_id = ?", blockRequest.UserID, blockRequest.BlockedID).First(&existingBlock).Error; err == nil {
		http.Error(w, "User is already blocked", http.StatusBadRequest)
		return
	}

	// Create the block entry
	block := entity.BlockedUser{
		UserID:    blockRequest.UserID,
		BlockedID: blockRequest.BlockedID,
	}
	if err := h.AuthService.DB.Create(&block).Error; err != nil {
		http.Error(w, "Error blocking user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User blocked successfully",
	})
}

func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	var unblockRequest struct {
		UserID    uint `json:"user_id"`    // The user who is unblocking
		BlockedID uint `json:"blocked_id"` // The user to unblock
	}
	if err := json.NewDecoder(r.Body).Decode(&unblockRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if unblockRequest.UserID == 0 || unblockRequest.BlockedID == 0 {
		http.Error(w, "user_id and blocked_id are required", http.StatusBadRequest)
		return
	}

	// Check if the block exists
	var block entity.BlockedUser
	if err := h.AuthService.DB.Where("user_id = ? AND blocked_id = ?", unblockRequest.UserID, unblockRequest.BlockedID).First(&block).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User is not blocked", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding block entry", http.StatusInternalServerError)
		return
	}

	// Delete the block entry (soft delete)
	if err := h.AuthService.DB.Delete(&block).Error; err != nil {
		http.Error(w, "Error unblocking user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User unblocked successfully",
	})
}
