package follow

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"social-network/internal/apperror"
	"social-network/internal/models"
	"social-network/internal/repository"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Generate a unique database name per test to ensure isolation.
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate random db name: %v", err)
	}
	dbName := hex.EncodeToString(bytes)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on&_journal_mode=WAL", dbName)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	// Restrict to 1 connection to prevent concurrency issues.
	db.SetMaxOpenConns(1)

	t.Cleanup(func() { db.Close() })

	if err := runMigrations(db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

func runMigrations(db *sql.DB) error {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationsPath := filepath.Join(basepath, "..", "db", "migrations", "sqlite")

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func TestSendFollowRequest(t *testing.T) {
	const (
		senderID = "sender-123"
		targetID = "target-456"
	)

	tests := []struct {
		name      string
		senderID  string
		targetID  string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:     "Cannot follow yourself",
			senderID: senderID,
			targetID: senderID,
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Already following",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com"})
				_ = follows.CreateFollower(senderID, targetID)
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrConflict) {
					t.Errorf("expected ErrConflict, got %v", err)
				}
			},
		},
		{
			name:     "Public profile auto-follow",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com", IsPublic: true})
			},
		},
		{
			name:     "Private profile creates follow request",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com", IsPublic: false})
			},
		},
		{
			name:     "Follow request already pending",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com", IsPublic: false})
				_ = follows.CreateFollowRequest(senderID, targetID)
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrConflict) {
					t.Errorf("expected ErrConflict, got %v", err)
				}
			},
		},
		{
			name:     "Target user not found",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Database error on IsFollowing",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com", IsPublic: true})
				_, _ = db.Exec("DROP TABLE followers")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
		{
			name:     "Database error on CreateFollower",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com", IsPublic: true})
				_, _ = db.Exec("DROP TABLE followers")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
		{
			name:     "Database error on CreateFollowRequest",
			senderID: senderID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: targetID, Email: "target@example.com", IsPublic: false})
				_, _ = db.Exec("DROP TABLE follow_requests")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, nil)
			err := svc.SendFollowRequest(tt.senderID, tt.targetID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAcceptRequest(t *testing.T) {
	const (
		senderID    = "sender-123"
		recipientID = "recipient-456"
	)

	tests := []struct {
		name        string
		senderID    string
		recipientID string
		setupDB     func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertErr   func(t *testing.T, err error)
	}{
		{
			name:        "Success",
			senderID:    senderID,
			recipientID: recipientID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: recipientID, Email: "recipient@example.com"})
				_ = follows.CreateFollowRequest(senderID, recipientID)
			},
		},
		{
			name:        "Pending request not found",
			senderID:    senderID,
			recipientID: recipientID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: recipientID, Email: "recipient@example.com"})
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:        "Database error on AcceptFollowRequest",
			senderID:    senderID,
			recipientID: recipientID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: recipientID, Email: "recipient@example.com"})
				_ = follows.CreateFollowRequest(senderID, recipientID)
				_, _ = db.Exec("DROP TABLE follow_requests")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, nil)
			err := svc.AcceptRequest(tt.recipientID, tt.senderID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDeclineRequest(t *testing.T) {
	const (
		senderID    = "sender-123"
		recipientID = "recipient-456"
	)

	tests := []struct {
		name        string
		senderID    string
		recipientID string
		setupDB     func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertErr   func(t *testing.T, err error)
	}{
		{
			name:        "Success",
			senderID:    senderID,
			recipientID: recipientID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: recipientID, Email: "recipient@example.com"})
				_ = follows.CreateFollowRequest(senderID, recipientID)
			},
		},
		{
			name:        "Pending request not found",
			senderID:    senderID,
			recipientID: recipientID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: recipientID, Email: "recipient@example.com"})
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:        "Database error",
			senderID:    senderID,
			recipientID: recipientID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: senderID, Email: "sender@example.com"})
				_ = users.CreateUser(&models.User{ID: recipientID, Email: "recipient@example.com"})
				_ = follows.CreateFollowRequest(senderID, recipientID)
				_, _ = db.Exec("DROP TABLE follow_requests")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, nil)
			err := svc.DeclineRequest(tt.recipientID, tt.senderID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUnfollow(t *testing.T) {
	const (
		followerID = "follower-123"
		followedID = "followed-456"
	)

	tests := []struct {
		name       string
		followerID string
		followedID string
		setupDB    func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertErr  func(t *testing.T, err error)
	}{
		{
			name:       "Cannot unfollow yourself",
			followerID: followerID,
			followedID: followerID,
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:       "Success",
			followerID: followerID,
			followedID: followedID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: followerID, Email: "follower@example.com"})
				_ = users.CreateUser(&models.User{ID: followedID, Email: "followed@example.com"})
				_ = follows.CreateFollower(followerID, followedID)
			},
		},
		{
			name:       "Follow relationship not found",
			followerID: followerID,
			followedID: followedID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: followerID, Email: "follower@example.com"})
				_ = users.CreateUser(&models.User{ID: followedID, Email: "followed@example.com"})
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:       "Database error",
			followerID: followerID,
			followedID: followedID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: followerID, Email: "follower@example.com"})
				_ = users.CreateUser(&models.User{ID: followedID, Email: "followed@example.com"})
				_ = follows.CreateFollower(followerID, followedID)
				_, _ = db.Exec("DROP TABLE followers")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, nil)
			err := svc.Unfollow(tt.followerID, tt.followedID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsFollowing(t *testing.T) {
	const (
		followerID = "follower-123"
		followedID = "followed-456"
	)

	tests := []struct {
		name       string
		followerID string
		followedID string
		setupDB    func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertRes  func(t *testing.T, isFollowing bool, err error)
	}{
		{
			name:       "Is following",
			followerID: followerID,
			followedID: followedID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: followerID, Email: "follower@example.com"})
				_ = users.CreateUser(&models.User{ID: followedID, Email: "followed@example.com"})
				_ = follows.CreateFollower(followerID, followedID)
			},
			assertRes: func(t *testing.T, isFollowing bool, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !isFollowing {
					t.Error("expected isFollowing to be true")
				}
			},
		},
		{
			name:       "Not following",
			followerID: followerID,
			followedID: followedID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: followerID, Email: "follower@example.com"})
				_ = users.CreateUser(&models.User{ID: followedID, Email: "followed@example.com"})
			},
			assertRes: func(t *testing.T, isFollowing bool, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if isFollowing {
					t.Error("expected isFollowing to be false")
				}
			},
		},
		{
			name:       "Database error",
			followerID: followerID,
			followedID: followedID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_, _ = db.Exec("DROP TABLE followers")
			},
			assertRes: func(t *testing.T, isFollowing bool, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, nil)
			isFollowing, err := svc.IsFollowing(tt.followerID, tt.followedID)

			tt.assertRes(t, isFollowing, err)
		})
	}
}

func TestGetPendingRequests(t *testing.T) {
	const userID = "user-123"

	tests := []struct {
		name      string
		userID    string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertRes func(t *testing.T, requests []*models.FollowRequest, err error)
	}{
		{
			name:   "Success with requests",
			userID: userID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: userID, Email: "user@example.com"})
				_ = users.CreateUser(&models.User{ID: "sender-1", Email: "sender1@example.com"})
				_ = users.CreateUser(&models.User{ID: "sender-2", Email: "sender2@example.com"})
				_ = follows.CreateFollowRequest("sender-1", userID)
				_ = follows.CreateFollowRequest("sender-2", userID)
			},
			assertRes: func(t *testing.T, requests []*models.FollowRequest, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(requests) != 2 {
					t.Errorf("expected 2 requests, got %d", len(requests))
				}
			},
		},
		{
			name:   "Success no requests",
			userID: userID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: userID, Email: "user@example.com"})
			},
			assertRes: func(t *testing.T, requests []*models.FollowRequest, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(requests) != 0 {
					t.Errorf("expected 0 requests, got %d", len(requests))
				}
			},
		},
		{
			name:   "Database error",
			userID: userID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: userID, Email: "user@example.com"})
				_, _ = db.Exec("DROP TABLE follow_requests")
			},
			assertRes: func(t *testing.T, requests []*models.FollowRequest, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, nil)
			requests, err := svc.GetPendingRequests(tt.userID)

			tt.assertRes(t, requests, err)
		})
	}
}
