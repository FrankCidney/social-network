package handlers

import (
	"net/http"
	"social-network/internal/middleware"
	"social-network/internal/websocket"
)

type WebSocketHandler struct {
	manager *websocket.Manager
}

func NewWebSocketHandler(manager *websocket.Manager) *WebSocketHandler {
	return &WebSocketHandler{manager: manager}
}

// ServeWS upgrades the HTTP connection to a WebSocket and registers the client
func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// RequireAuth middleware guarantees User is in context
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	h.manager.ServeWS(w, r, user.ID)
}
