package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received WebSocket request for path: %s", r.URL.Path)
	vars := mux.Vars(r)
	userIDStr := vars["user_id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Invalid user ID: %s", userIDStr)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	h.WebSocketService.HandleConnections(w, r, uint(userID))
}
