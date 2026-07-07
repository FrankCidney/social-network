package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

type FollowRepository interface {
	CreateFollowRequest(senderID, targetID string) error
	GetFollowRequest(senderID, targetID string) (*models.FollowRequest, error)
	AcceptFollowRequest(senderID, targetID string) error
	DeclineFollowRequest(senderID, targetID string) error
	CreateFollower(followerID, followedID string) error // used for auto-follow (public profiles)
	DeleteFollower(followerID, followedID string) error
	IsFollowing(followerID, followedID string) (bool, error)
	GetFollowers(userID string, limit, offset int) ([]*models.User, error)
	GetFollowing(userID string, limit, offset int) ([]*models.User, error)
	GetFollowerCount(userID string) (int, error)
	GetFollowingCount(userID string) (int, error)
	GetPendingRequests(userID string) ([]*models.FollowRequest, error)
}

type sqliteFollowRepo struct {
	db *sql.DB
}

func NewFollowRepository(db *sql.DB) FollowRepository {
	return &sqliteFollowRepo{db: db}
}

func (r *sqliteFollowRepo) CreateFollowRequest(senderID, targetID string) error {
	const query = `
		INSERT INTO follow_requests (sender_id, receiver_id, status, created_at)
		VALUES (?, ?, 'pending', datetime('now'))`
 
	_, err := r.db.Exec(query, senderID, targetID)
	if err != nil {
		if isSQLiteUniqueViolation(err) {
			return apperror.Conflict("follow request already exists")
		}
		return fmt.Errorf("create follow request: %w", err)
	}
	return nil
}

func (r *sqliteFollowRepo) GetFollowRequest(senderID, targetID string) (*models.FollowRequest, error) {
	const query = `
		SELECT sender_id, receiver_id, status, created_at
		FROM follow_requests WHERE sender_id = ? AND receiver_id = ?`
 
	fr := &models.FollowRequest{}
	err := r.db.QueryRow(query, senderID, targetID).Scan(
		&fr.SenderID, &fr.ReceiverID, &fr.Status, &fr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("follow request not found")
		}
		return nil, fmt.Errorf("get follow request: %w", err)
	}
	return fr, nil
}

// AcceptFollowRequest updates the request to 'accepted' and writes to followers in a single database transaction.
func (r *sqliteFollowRepo) AcceptFollowRequest(senderID, targetID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
 
	const updateReq = `
		UPDATE follow_requests SET status = 'accepted'
		WHERE sender_id = ? AND receiver_id = ? AND status = 'pending'`
 
	res, err := tx.Exec(updateReq, senderID, targetID)
	if err != nil {
		return fmt.Errorf("accept follow request update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Either the request doesn't exist or it's already accepted/declined.
		return apperror.NotFound("pending follow request not found")
	}
 
	// "IGNORE" is used for idempotency, so as to not throw an error if a user say clicks
	// "Accept Follow Request" several times in quick succession
	const insertFollower = `
		INSERT OR IGNORE INTO followers (follower_id, followed_id, created_at)
		VALUES (?, ?, datetime('now'))`
 
	if _, err = tx.Exec(insertFollower, senderID, targetID); err != nil {
		return fmt.Errorf("insert follower on accept: %w", err)
	}
 
	return tx.Commit()
}

func (r *sqliteFollowRepo) DeclineFollowRequest(senderID, targetID string) error {
	const query = `
		DELETE FROM follow_requests
		WHERE sender_id = ? AND receiver_id = ? AND status = 'pending'`
 
	res, err := r.db.Exec(query, senderID, targetID)
	if err != nil {
		return fmt.Errorf("decline follow request: %w", err)
	}
	return requireOneRow(res, "pending follow request")
}
 
func (r *sqliteFollowRepo) CreateFollower(followerID, followedID string) error {
	// "IGNORE" is used for idempotency, so as to not error should a user click follow
	// multiple times in quick succession
	const query = `
		INSERT OR IGNORE INTO followers (follower_id, followed_id, created_at)
		VALUES (?, ?, datetime('now'))`
 
	_, err := r.db.Exec(query, followerID, followedID)
	if err != nil {
		return fmt.Errorf("create follower: %w", err)
	}
	return nil
}

func (r *sqliteFollowRepo) DeleteFollower(followerID, followedID string) error {
	const query = `DELETE FROM followers WHERE follower_id = ? AND followed_id = ?`
	res, err := r.db.Exec(query, followerID, followedID)
	if err != nil {
		return fmt.Errorf("delete follower: %w", err)
	}
	return requireOneRow(res, "follow relationship")
}

func (r *sqliteFollowRepo) IsFollowing(followerID, followedID string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM followers WHERE follower_id = ? AND followed_id = ?
		)`
 
	var exists bool
	err := r.db.QueryRow(query, followerID, followedID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is following: %w", err)
	}
	return exists, nil
}

func (r *sqliteFollowRepo) GetFollowers(userID string, limit, offset int) ([]*models.User, error) {
	const query = `
		SELECT u.id, u.email, u.first_name, u.last_name, u.dob,
		       COALESCE(u.nickname, ''), COALESCE(u.about_me, ''), COALESCE(u.avatar_path, ''),
		       u.is_public, u.created_at
		FROM followers f
		JOIN users u ON u.id = f.follower_id
		WHERE f.followed_id = ?
		ORDER BY f.created_at DESC
		LIMIT ? OFFSET ?`
 
	return r.scanUsers(query, userID, limit, offset)
}

func (r *sqliteFollowRepo) GetFollowing(userID string, limit, offset int) ([]*models.User, error) {
	const query = `
		SELECT u.id, u.email, u.first_name, u.last_name, u.dob,
		       COALESCE(u.nickname, ''), COALESCE(u.about_me, ''), COALESCE(u.avatar_path, ''),
		       u.is_public, u.created_at
		FROM followers f
		JOIN users u ON u.id = f.followed_id
		WHERE f.follower_id = ?
		ORDER BY f.created_at DESC
		LIMIT ? OFFSET ?`
 
	return r.scanUsers(query, userID, limit, offset)
}

func (r *sqliteFollowRepo) GetFollowerCount(userID string) (int, error) {
	return r.countWhere("followers", "followed_id", userID)
}
 
func (r *sqliteFollowRepo) GetFollowingCount(userID string) (int, error) {
	return r.countWhere("followers", "follower_id", userID)
}

func (r *sqliteFollowRepo) GetPendingRequests(userID string) ([]*models.FollowRequest, error) {
	const query = `
		SELECT sender_id, receiver_id, status, created_at
		FROM follow_requests
		WHERE receiver_id = ? AND status = 'pending'
		ORDER BY created_at DESC`
 
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get pending requests: %w", err)
	}
	defer rows.Close()
 
	var requests []*models.FollowRequest
	for rows.Next() {
		fr := &models.FollowRequest{}
		if err := rows.Scan(&fr.SenderID, &fr.ReceiverID, &fr.Status, &fr.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan follow request: %w", err)
		}
		requests = append(requests, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return requests, nil
}

func (r *sqliteFollowRepo) scanUsers(query, userID string, limit, offset int) ([]*models.User, error) {
	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()
 
	var users []*models.User
	for rows.Next() {
		u := &models.User{}

		if err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.DOB,
			&u.Nickname, &u.AboutMe, &u.AvatarPath, &u.IsPublic, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return users, nil
}

func (r *sqliteFollowRepo) countWhere(table, column, value string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column)
	var count int
	if err := r.db.QueryRow(query, value).Scan(&count); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return count, nil
}
