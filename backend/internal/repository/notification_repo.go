package repository

import (
	"database/sql"
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

type NotificationRepository interface {
	CreateNotification(notification *models.Notification) error
	GetUserNotifications(userID string, limit, offset int) ([]*models.Notification, error)
	GetNotificationCount(userID string) (int, error)
	MarkNotificationAsRead(userID, notificationID string) error
	MarkNotificationAsResolved(userID, notificationID string) error
	MarkAllNotificationsAsRead(userID string) error
}

type sqliteNotificationRepo struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &sqliteNotificationRepo{db: db}
}

func (r *sqliteNotificationRepo) CreateNotification(notification *models.Notification) error {
	const query = `
		INSERT INTO notifications (id, user_id, actor_id, type, group_id, event_id, is_read, is_resolved, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
		query,
		notification.ID,
		notification.UserID,
		notification.ActorID,
		notification.Type,
		notification.GroupID,
		notification.EventID,
		notification.IsRead,
		notification.IsResolved,
		notification.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}

func (r *sqliteNotificationRepo) GetUserNotifications(userID string, limit, offset int) ([]*models.Notification, error) {
	const query = `
		SELECT
			n.id,
			n.user_id,
			n.actor_id,
			n.type,
			n.group_id,
			n.event_id,
			n.is_read,
			n.is_resolved,
			n.created_at,
			u.id,
			u.first_name,
			u.last_name,
			COALESCE(u.nickname, ''),
			COALESCE(u.avatar_path, ''),
			u.is_public
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		WHERE n.user_id = ?
		ORDER BY n.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get user notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		notification := &models.Notification{}
		actor := &models.PublicUser{}
		var actorID sql.NullString
		var actorFirstName sql.NullString
		var actorLastName sql.NullString
		var actorNickname sql.NullString
		var actorAvatarPath sql.NullString
		var actorIsPublic sql.NullBool

		if err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.ActorID,
			&notification.Type,
			&notification.GroupID,
			&notification.EventID,
			&notification.IsRead,
			&notification.IsResolved,
			&notification.CreatedAt,
			&actorID,
			&actorFirstName,
			&actorLastName,
			&actorNickname,
			&actorAvatarPath,
			&actorIsPublic,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}

		if actorID.Valid {
			actor.ID = actorID.String
			actor.FirstName = actorFirstName.String
			actor.LastName = actorLastName.String
			actor.Nickname = actorNickname.String
			actor.AvatarPath = actorAvatarPath.String
			actor.IsPublic = actorIsPublic.Bool
			notification.Actor = actor
		}

		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, nil
}

func (r *sqliteNotificationRepo) GetNotificationCount(userID string) (int, error) {
	const query = `SELECT COUNT(*) FROM notifications WHERE user_id = ?`

	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count notifications: %w", err)
	}

	return count, nil
}

func (r *sqliteNotificationRepo) MarkNotificationAsRead(userID, notificationID string) error {
	const query = `UPDATE notifications SET is_read = 1 WHERE id = ? AND user_id = ?`

	result, err := r.db.Exec(query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("mark notification as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("notification read rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return apperror.NotFound("notification not found")
	}

	return nil
}

func (r *sqliteNotificationRepo) MarkNotificationAsResolved(userID, notificationID string) error {
	const query = `UPDATE notifications SET is_resolved = 1, is_read = 1 WHERE id = ? AND user_id = ?`

	result, err := r.db.Exec(query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("mark notification as resolved: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("notification resolved rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return apperror.NotFound("notification not found")
	}

	return nil
}

func (r *sqliteNotificationRepo) MarkAllNotificationsAsRead(userID string) error {
	const query = `UPDATE notifications SET is_read = 1 WHERE user_id = ?`

	if _, err := r.db.Exec(query, userID); err != nil {
		return fmt.Errorf("mark all notifications as read: %w", err)
	}

	return nil
}
