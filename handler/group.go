package handler

import (
	"chat_app/entity"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var group struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if group.Name == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	newGroup := entity.Group{Name: group.Name}
	if err := h.GroupService.DB.Create(&newGroup).Error; err != nil {
		http.Error(w, "Error creating group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   newGroup.ID,
		"name": newGroup.Name,
	})
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	var groups []entity.Group
	if err := h.GroupService.DB.Find(&groups).Error; err != nil {
		http.Error(w, "Error fetching groups", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

func (h *Handler) HandleGroupMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupIDStr := vars["group_id"]
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.AddUserToGroup(w, r, uint(groupID))
	case http.MethodGet:
		h.ListGroupMembers(w, r, uint(groupID))
	case http.MethodDelete:
		h.RemoveUserFromGroup(w, r, uint(groupID))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) AddUserToGroup(w http.ResponseWriter, r *http.Request, groupID uint) {
	var member struct {
		UserID uint `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&member); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if member.UserID == 0 {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Check if the group exists
	var group entity.Group
	if err := h.GroupService.DB.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding group", http.StatusInternalServerError)
		return
	}

	// Check if the user exists
	var user entity.User
	if err := h.GroupService.DB.First(&user, member.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding user", http.StatusInternalServerError)
		return
	}

	// Check if the user is already in the group
	var existingMember entity.GroupMember
	if err := h.GroupService.DB.Where("group_id = ? AND user_id = ? AND deleted_at IS NULL", groupID, member.UserID).First(&existingMember).Error; err == nil {
		http.Error(w, "User is already in the group", http.StatusBadRequest)
		return
	}

	// Add the user to the group
	groupMember := entity.GroupMember{
		GroupID: groupID,
		UserID:  member.UserID,
	}
	if err := h.GroupService.DB.Create(&groupMember).Error; err != nil {
		http.Error(w, "Error adding user to group", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"group_id": groupID,
		"user_id":  member.UserID,
	})
}

func (h *Handler) ListGroupMembers(w http.ResponseWriter, r *http.Request, groupID uint) {
	var members []entity.GroupMember
	if err := h.GroupService.DB.Where("group_id = ? AND deleted_at IS NULL", groupID).Find(&members).Error; err != nil {
		http.Error(w, "Error fetching group members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *Handler) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request, groupID uint) {
	var member struct {
		UserID uint `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&member); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if member.UserID == 0 {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Check if the group exists
	var group entity.Group
	if err := h.GroupService.DB.First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding group", http.StatusInternalServerError)
		return
	}

	// Check if the user is in the group
	var groupMember entity.GroupMember
	if err := h.GroupService.DB.Where("group_id = ? AND user_id = ? AND deleted_at IS NULL", groupID, member.UserID).First(&groupMember).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User is not in the group", http.StatusNotFound)
			return
		}
		http.Error(w, "Error finding group member", http.StatusInternalServerError)
		return
	}

	// Soft delete the group member
	if err := h.GroupService.DB.Delete(&groupMember).Error; err != nil {
		http.Error(w, "Error removing user from group", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User removed from group",
	})
}
