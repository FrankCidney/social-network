package db

import (
	"context"
	"social-network/internal/models"
)

// CreateNotification inserts a new notification
func (s *SQLiteStore) CreateNotification(ctx context.Context, notif *models.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, actor_id, type, group_id, event_id, is_read, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query,
		notif.ID,
		notif.UserID,
		notif.ActorID,
		notif.Type,
		notif.GroupID,
		notif.EventID,
	)
	return err
}

// GetUserNotifications retrieves all notifications for a specific user
func (s *SQLiteStore) GetUserNotifications(ctx context.Context, userID string) ([]*models.Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, group_id, event_id, is_read, created_at
		FROM notifications
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := s.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		n := &models.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type, &n.GroupID, &n.EventID, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

// MarkNotificationAsRead marks a specific notification as read
func (s *SQLiteStore) MarkNotificationAsRead(ctx context.Context, notificationID string) error {
	query := `UPDATE notifications SET is_read = 1 WHERE id = ?`
	_, err := s.ExecContext(ctx, query, notificationID)
	return err
}
