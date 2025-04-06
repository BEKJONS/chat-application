package handler

import "chat_app/services"

type Handler struct {
	GroupService     *services.GroupService
	WebSocketService *services.WebSocketService
	AuthService      *services.AuthService
}

func NewHandler(authService *services.AuthService, groupService *services.GroupService, wsService *services.WebSocketService) *Handler {
	return &Handler{
		AuthService:      authService,
		GroupService:     groupService,
		WebSocketService: wsService,
	}
}
