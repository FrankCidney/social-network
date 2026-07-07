package db

import (
	"context"
	"social-network/internal/models"
)

// SaveMessage stores a new private or group chat message
func (s *SQLiteStore) SaveMessage(ctx context.Context, msg *models.Message) error {
	query := `
		INSERT INTO messages (id, sender_id, receiver_id, group_id, content, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query,
		msg.ID,
		msg.SenderID,
		msg.ReceiverID,
		msg.GroupID,
		msg.Content,
	)
	return err
}

// GetPrivateMessages retrieves messages between two users
func (s *SQLiteStore) GetPrivateMessages(ctx context.Context, user1ID, user2ID string) ([]*models.Message, error) {
	query := `
		SELECT id, sender_id, receiver_id, group_id, content, created_at
		FROM messages
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
		ORDER BY created_at ASC
	`
	rows, err := s.QueryContext(ctx, query, user1ID, user2ID, user2ID, user1ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.GroupID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// GetGroupMessages retrieves messages for a specific group
func (s *SQLiteStore) GetGroupMessages(ctx context.Context, groupID string) ([]*models.Message, error) {
	query := `
		SELECT id, sender_id, receiver_id, group_id, content, created_at
		FROM messages
		WHERE group_id = ?
		ORDER BY created_at ASC
	`
	rows, err := s.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.GroupID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
