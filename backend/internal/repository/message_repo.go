package repository

import (
	"database/sql"
	"fmt"

	"social-network/internal/models"
)

type MessageRepository interface {
	CreateMessage(msg *models.Message) error
	GetPrivateMessages(user1ID, user2ID string, limit, offset int) ([]*models.Message, error)
	GetGroupMessages(groupID string, limit, offset int) ([]*models.Message, error)
	GetConversations(userID string) ([]*models.Conversation, error)
	MarkConversationRead(userID, otherUserID string) error
}

type sqliteMessageRepo struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) MessageRepository {
	return &sqliteMessageRepo{db: db}
}

func (r *sqliteMessageRepo) CreateMessage(msg *models.Message) error {
	const query = `
		INSERT INTO messages (id, sender_id, receiver_id, group_id, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, msg.ID, msg.SenderID, msg.ReceiverID, msg.GroupID, msg.Content, msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func (r *sqliteMessageRepo) GetPrivateMessages(user1ID, user2ID string, limit, offset int) ([]*models.Message, error) {
	const query = `
		SELECT id, sender_id, receiver_id, content, created_at
		FROM messages
		WHERE group_id IS NULL AND 
		      ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, user1ID, user2ID, user2ID, user1ID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get private messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *sqliteMessageRepo) GetGroupMessages(groupID string, limit, offset int) ([]*models.Message, error) {
	const query = `
		SELECT id, sender_id, group_id, content, created_at
		FROM messages
		WHERE group_id = ?
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, groupID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get group messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.GroupID, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// ADD THIS
func (r *sqliteMessageRepo) GetConversations(userID string) ([]*models.Conversation, error) {
	const partnersQuery = `
		SELECT
			CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END AS partner_id,
			MAX(created_at) AS last_created
		FROM messages
		WHERE group_id IS NULL AND (sender_id = ? OR receiver_id = ?)
		GROUP BY partner_id
		ORDER BY last_created DESC`

	rows, err := r.db.Query(partnersQuery, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("get conversation partners: %w", err)
	}
	defer rows.Close()

	var partnerIDs []string
	for rows.Next() {
		var partnerID, lastCreated string
		if err := rows.Scan(&partnerID, &lastCreated); err != nil {
			return nil, fmt.Errorf("scan conversation partner: %w", err)
		}
		partnerIDs = append(partnerIDs, partnerID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation partners: %w", err)
	}

	conversations := make([]*models.Conversation, 0, len(partnerIDs))
	for _, partnerID := range partnerIDs {
		conv, err := r.buildConversation(userID, partnerID)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func (r *sqliteMessageRepo) buildConversation(userID, partnerID string) (*models.Conversation, error) {
	const userQuery = `
		SELECT id, first_name, last_name, nickname, avatar_path, is_public
		FROM users
		WHERE id = ?`

	user := &models.PublicUser{}
	var nickname, avatarPath sql.NullString
	if err := r.db.QueryRow(userQuery, partnerID).Scan(
		&user.ID, &user.FirstName, &user.LastName, &nickname, &avatarPath, &user.IsPublic,
	); err != nil {
		return nil, fmt.Errorf("get conversation partner user: %w", err)
	}
	user.Nickname = nickname.String
	user.AvatarPath = avatarPath.String

	const lastMsgQuery = `
		SELECT id, sender_id, receiver_id, content, created_at, read_at
		FROM messages
		WHERE group_id IS NULL AND
		      ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
		ORDER BY created_at DESC
		LIMIT 1`

	lastMsg := &models.Message{}
	var readAt sql.NullTime
	if err := r.db.QueryRow(lastMsgQuery, userID, partnerID, partnerID, userID).Scan(
		&lastMsg.ID, &lastMsg.SenderID, &lastMsg.ReceiverID, &lastMsg.Content, &lastMsg.CreatedAt, &readAt,
	); err != nil {
		return nil, fmt.Errorf("get last message: %w", err)
	}
	if readAt.Valid {
		lastMsg.ReadAt = &readAt.Time
	}

	const unreadQuery = `
		SELECT COUNT(*)
		FROM messages
		WHERE group_id IS NULL AND sender_id = ? AND receiver_id = ? AND read_at IS NULL`

	var unreadCount int
	if err := r.db.QueryRow(unreadQuery, partnerID, userID).Scan(&unreadCount); err != nil {
		return nil, fmt.Errorf("get unread count: %w", err)
	}

	return &models.Conversation{User: user, LastMessage: lastMsg, UnreadCount: unreadCount}, nil
}

func (r *sqliteMessageRepo) MarkConversationRead(userID, otherUserID string) error {
	const query = `
		UPDATE messages
		SET read_at = CURRENT_TIMESTAMP
		WHERE group_id IS NULL AND receiver_id = ? AND sender_id = ? AND read_at IS NULL`

	if _, err := r.db.Exec(query, userID, otherUserID); err != nil {
		return fmt.Errorf("mark conversation read: %w", err)
	}
	return nil
}
