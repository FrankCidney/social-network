package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Manager tracks active WebSocket connections and broadcasts messages
type Manager struct {
	clients     map[*Client]bool
	userClients map[string]map[*Client]bool // Map user ID to their active clients
	sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		clients:     make(map[*Client]bool),
		userClients: make(map[string]map[*Client]bool),
	}
}

func (m *Manager) AddClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
	if _, ok := m.userClients[client.UserID]; !ok {
		m.userClients[client.UserID] = make(map[*Client]bool)
	}
	m.userClients[client.UserID][client] = true
	slog.Info("websocket client connected", "userID", client.UserID)
}

func (m *Manager) RemoveClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.conn.Close()
		delete(m.clients, client)
		if userMap, ok := m.userClients[client.UserID]; ok {
			delete(userMap, client)
			if len(userMap) == 0 {
				delete(m.userClients, client.UserID)
			}
		}
		slog.Info("websocket client disconnected", "userID", client.UserID)
	}
}

// WSPayload represents the JSON structure sent over the WebSocket
type WSPayload struct {
	Type    string      `json:"type"` // e.g., "notification", "chat_message"
	Payload interface{} `json:"payload"`
}

// BroadcastToUser sends a payload to all active connections of a specific user
func (m *Manager) BroadcastToUser(userID string, payloadType string, data interface{}) {
	m.RLock()
	defer m.RUnlock()

	userMap, ok := m.userClients[userID]
	if !ok {
		return // User not connected
	}

	payload := WSPayload{
		Type:    payloadType,
		Payload: data,
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal websocket payload", "error", err)
		return
	}

	for client := range userMap {
		select {
		case client.send <- msg:
		default:
			go m.RemoveClient(client)
		}
	}
}

// BroadcastToGroup sends a payload to a list of users (e.g. group members)
func (m *Manager) BroadcastToGroup(userIDs []string, payloadType string, data interface{}) {
	for _, userID := range userIDs {
		m.BroadcastToUser(userID, payloadType, data)
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade connection", "error", err)
		return
	}

	client := &Client{
		manager: m,
		conn:    conn,
		send:    make(chan []byte, 256),
		UserID:  userID,
	}

	m.AddClient(client)

	go client.readPump()
	go client.writePump()
}
