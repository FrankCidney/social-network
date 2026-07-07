package models

import "time"

// Message represents a direct or group message
type Message struct {
	ID         string     `json:"id"`
	SenderID   string     `json:"sender_id"`
	ReceiverID *string    `json:"receiver_id,omitempty"` // Nullable for group chat
	GroupID    *string    `json:"group_id,omitempty"`    // Nullable for private chat
	Content    string     `json:"content"`
	CreatedAt  time.Time  `json:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	Author     *User      `json:"author,omitempty"`
}

// Notification represents a user notification
type Notification struct {
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	ActorID    string      `json:"actor_id"`
	Type       string      `json:"type"`
	GroupID    *string     `json:"group_id,omitempty"`
	EventID    *string     `json:"event_id,omitempty"`
	IsRead     bool        `json:"is_read"`
	IsResolved bool        `json:"is_resolved"`
	CreatedAt  time.Time   `json:"created_at"`
	Actor      *PublicUser `json:"actor,omitempty"`
}

type Conversation struct {
	User        *PublicUser `json:"user"`
	LastMessage *Message    `json:"last_message,omitempty"`
	UnreadCount int         `json:"unread_count"`
}
