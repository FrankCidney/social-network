package websocket

import "social-network/internal/models"

// Notifier defines the interface for dispatching real-time notifications
type Notifier interface {
	NotifyUser(userID string, notification *models.Notification)
	NotifyGroup(userIDs []string, notification *models.Notification)
	NotifyMessage(userID string, msg *models.Message)
	NotifyGroupMessage(userIDs []string, msg *models.Message)
}

// WSNotifier implements the Notifier interface using the WebSocket Manager
type WSNotifier struct {
	manager *Manager
}

func NewWSNotifier(manager *Manager) *WSNotifier {
	return &WSNotifier{manager: manager}
}

func (n *WSNotifier) NotifyUser(userID string, notification *models.Notification) {
	n.manager.BroadcastToUser(userID, "notification", notification)
}

func (n *WSNotifier) NotifyGroup(userIDs []string, notification *models.Notification) {
	n.manager.BroadcastToGroup(userIDs, "notification", notification)
}

func (n *WSNotifier) NotifyMessage(userID string, msg *models.Message) {
	n.manager.BroadcastToUser(userID, "chat_message", msg)
}

func (n *WSNotifier) NotifyGroupMessage(userIDs []string, msg *models.Message) {
	n.manager.BroadcastToGroup(userIDs, "chat_message", msg)
}

func (n *WSNotifier) NotifyFollowRequest(senderID, receiverID string) error {
	n.manager.BroadcastToUser(receiverID, "notification", &models.Notification{
		Type:    "follow_request",
		ActorID: senderID,
		UserID:  receiverID,
	})
	return nil
}

func (n *WSNotifier) NotifyFollowAccepted(senderID, receiverID string) error {
	n.manager.BroadcastToUser(senderID, "notification", &models.Notification{
		Type:    "follow_accepted",
		ActorID: receiverID,
		UserID:  senderID,
	})
	return nil
}
