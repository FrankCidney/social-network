package repository

import (
	"errors"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"testing"
)

func TestCreateSession(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository)
		session *models.Session
		wantErr bool
	}{
		{
			name: "success_create_session",
			setup: func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {
				u := &models.User{
					ID:    "user-1",
					Email: "user1@example.com",
				}
				if err := userRepo.CreateUser(u); err != nil {
					t.Fatalf("failed to create user: %v", err)
				}
			},
			session: &models.Session{
				Token:     "token-1",
				UserID:    "user-1",
				CreatedAt: "2026-06-30T12:00:00Z",
				ExpiresAt: "2026-06-30T13:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "failure_foreign_key_violation",
			setup: func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {
				// No user created
			},
			session: &models.Session{
				Token:     "token-2",
				UserID:    "non-existent-user",
				CreatedAt: "2026-06-30T12:00:00Z",
				ExpiresAt: "2026-06-30T13:00:00Z",
			},
			wantErr: true, // Should fail due to FOREIGN KEY constraint
		},
		{
			name: "failure_duplicate_token",
			setup: func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {
				u := &models.User{
					ID:    "user-1",
					Email: "user1@example.com",
				}
				if err := userRepo.CreateUser(u); err != nil {
					t.Fatalf("failed to create user: %v", err)
				}
				s := &models.Session{
					Token:     "token-1",
					UserID:    "user-1",
					CreatedAt: "2026-06-30T12:00:00Z",
					ExpiresAt: "2026-06-30T13:00:00Z",
				}
				if err := sessionRepo.CreateSession(s); err != nil {
					t.Fatalf("failed to create initial session: %v", err)
				}
			},
			session: &models.Session{
				Token:     "token-1", // duplicate token
				UserID:    "user-1",
				CreatedAt: "2026-06-30T12:00:00Z",
				ExpiresAt: "2026-06-30T13:00:00Z",
			},
			wantErr: true, // Should fail due to UNIQUE constraint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			sessionRepo := NewSessionRepository(db)

			tt.setup(t, userRepo, sessionRepo)

			err := sessionRepo.CreateSession(tt.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if err == nil {
				// Verify it can be retrieved
				got, err := sessionRepo.GetSessionByToken(tt.session.Token)
				if err != nil {
					t.Errorf("failed to retrieve created session: %v", err)
				}
				if got.Token != tt.session.Token || got.UserID != tt.session.UserID {
					t.Errorf("retrieved session %+v does not match created %+v", got, tt.session)
				}
			}
		})
	}
}

func TestGetSessionByToken(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository)
		token   string
		want    *models.Session
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {
				u := &models.User{
					ID:    "user-1",
					Email: "user1@example.com",
				}
				if err := userRepo.CreateUser(u); err != nil {
					t.Fatalf("failed to create user: %v", err)
				}
				s := &models.Session{
					Token:     "token-1",
					UserID:    "user-1",
					CreatedAt: "2026-06-30T12:00:00Z",
					ExpiresAt: "2026-06-30T13:00:00Z",
				}
				if err := sessionRepo.CreateSession(s); err != nil {
					t.Fatalf("failed to create session: %v", err)
				}
			},
			token: "token-1",
			want: &models.Session{
				Token:     "token-1",
				UserID:    "user-1",
				CreatedAt: "2026-06-30T12:00:00Z",
				ExpiresAt: "2026-06-30T13:00:00Z",
			},
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {},
			token:   "non-existent-token",
			want:    nil,
			wantErr: apperror.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			sessionRepo := NewSessionRepository(db)

			tt.setup(t, userRepo, sessionRepo)

			got, err := sessionRepo.GetSessionByToken(tt.token)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if got.Token != tt.want.Token || got.UserID != tt.want.UserID {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestDeleteSession(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository)
		token string
	}{
		{
			name: "delete_existing_session",
			setup: func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {
				u := &models.User{
					ID:    "user-1",
					Email: "user1@example.com",
				}
				if err := userRepo.CreateUser(u); err != nil {
					t.Fatalf("failed to create user: %v", err)
				}
				s := &models.Session{
					Token:     "token-1",
					UserID:    "user-1",
					CreatedAt: "2026-06-30T12:00:00Z",
					ExpiresAt: "2026-06-30T13:00:00Z",
				}
				if err := sessionRepo.CreateSession(s); err != nil {
					t.Fatalf("failed to create session: %v", err)
				}
			},
			token: "token-1",
		},
		{
			name:  "delete_non_existent_session_does_not_error",
			setup: func(t *testing.T, userRepo UserRepository, sessionRepo SessionRepository) {},
			token: "non-existent-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			sessionRepo := NewSessionRepository(db)

			tt.setup(t, userRepo, sessionRepo)

			err := sessionRepo.DeleteSession(tt.token)
			if err != nil {
				t.Errorf("DeleteSession() unexpected error: %v", err)
			}

			// Verify it's gone
			_, err = sessionRepo.GetSessionByToken(tt.token)
			if !errors.Is(err, apperror.ErrUnauthorized) {
				t.Errorf("expected session to be gone (ErrUnauthorized), got error: %v", err)
			}
		})
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)

	// Seed user
	u := &models.User{
		ID:    "user-1",
		Email: "user1@example.com",
	}
	if err := userRepo.CreateUser(u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 1. Expired session (using a far past date to ensure it is smaller than datetime('now'))
	expiredSession := &models.Session{
		Token:     "expired-token",
		UserID:    "user-1",
		CreatedAt: "2000-01-01T00:00:00Z",
		ExpiresAt: "2000-01-01T01:00:00Z",
	}
	if err := sessionRepo.CreateSession(expiredSession); err != nil {
		t.Fatalf("failed to create expired session: %v", err)
	}

	// 2. Valid session (using a far future date to ensure it is larger than datetime('now'))
	validSession := &models.Session{
		Token:     "valid-token",
		UserID:    "user-1",
		CreatedAt: "2030-01-01T00:00:00Z",
		ExpiresAt: "2030-01-01T01:00:00Z",
	}
	if err := sessionRepo.CreateSession(validSession); err != nil {
		t.Fatalf("failed to create valid session: %v", err)
	}

	// Delete expired sessions
	if err := sessionRepo.DeleteExpiredSessions(); err != nil {
		t.Fatalf("DeleteExpiredSessions() unexpected error: %v", err)
	}

	// Verify expired session is gone
	_, err := sessionRepo.GetSessionByToken("expired-token")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected expired session to be deleted, got error: %v", err)
	}

	// Verify valid session is still there
	gotValid, err := sessionRepo.GetSessionByToken("valid-token")
	if err != nil {
		t.Errorf("expected valid session to still exist, got error: %v", err)
	}
	if gotValid.Token != "valid-token" {
		t.Errorf("expected valid session token to match, got: %s", gotValid.Token)
	}
}
