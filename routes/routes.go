package routes

import (
	"chat_app/handler"
	"chat_app/services"
	"log"

	"github.com/gorilla/mux"
)

type Routes struct {
	Handler *handler.Handler
}

func NewRoutes(authService *services.AuthService, groupService *services.GroupService, wsService *services.WebSocketService) *Routes {
	return &Routes{
		Handler: handler.NewHandler(authService, groupService, wsService),
	}
}

func (r *Routes) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// WebSocket route
	router.HandleFunc("/ws/{user_id}", r.Handler.HandleWebSocket).Methods("GET")

	// Auth routes
	router.HandleFunc("/register", r.Handler.Register).Methods("POST")
	router.HandleFunc("/login", r.Handler.Login).Methods("POST")
	router.HandleFunc("/block", r.Handler.BlockUser).Methods("POST")
	router.HandleFunc("/unblock", r.Handler.UnblockUser).Methods("POST")

	// Group routes
	router.HandleFunc("/groups", r.Handler.CreateGroup).Methods("POST")
	router.HandleFunc("/groups", r.Handler.ListGroups).Methods("GET")
	router.HandleFunc("/groups/{group_id}/members", r.Handler.HandleGroupMembers).Methods("POST", "GET", "DELETE")

	log.Println("Routes set up successfully")
	return router
}
