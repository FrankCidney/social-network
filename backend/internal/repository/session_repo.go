package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

type SessionRepository interface {
	CreateSession(s *models.Session) error
	GetSessionByToken(token string) (*models.Session, error)
	DeleteSession(token string) error
	DeleteExpiredSessions() error
}

type sqliteSessionRepo struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sqliteSessionRepo{db: db}
}

func (r *sqliteSessionRepo) CreateSession(s *models.Session) error {
	const query = `
		INSERT INTO sessions (token, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?)`

	_, err := r.db.Exec(query, s.Token, s.UserID, s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (r *sqliteSessionRepo) GetSessionByToken(token string) (*models.Session, error) {
	const query = `
		SELECT token, user_id, created_at, expires_at
		FROM sessions WHERE token = ?`

	s := &models.Session{}
	err := r.db.QueryRow(query, token).Scan(
		&s.Token, &s.UserID, &s.CreatedAt, &s.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.Unauthorized("session not found")
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	return s, nil
}

func (r *sqliteSessionRepo) DeleteSession(token string) error {
	const query = `DELETE FROM sessions WHERE token = ?`

	// Exec doesn't error if the session doesn't exist
	_, err := r.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (r *sqliteSessionRepo) DeleteExpiredSessions() error {
	const query = `DELETE FROM sessions WHERE expires_at < datetime('now')`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}
