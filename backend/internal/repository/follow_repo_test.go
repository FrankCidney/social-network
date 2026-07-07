package repository

import (
	"errors"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"testing"
)

func createTestUser(t *testing.T, repo UserRepository, id, email string) {
	t.Helper()
	u := &models.User{
		ID:        id,
		Email:     email,
		FirstName: "FN-" + id,
		LastName:  "LN-" + id,
		DOB:       "1990-01-01T00:00:00Z",
	}
	if err := repo.CreateUser(u); err != nil {
		t.Fatalf("failed to create user %s: %v", id, err)
	}
}

func TestCreateFollowRequest(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository)
		sender   string
		receiver string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
			},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  nil,
		},
		{
			name: "conflict_duplicate_request",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
				if err := followRepo.CreateFollowRequest("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed to create request: %v", err)
				}
			},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  apperror.ErrConflict,
		},
		{
			name: "failure_foreign_key_violation",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				// No users created
			},
			sender:   "non-existent-1",
			receiver: "non-existent-2",
			wantErr:  errors.New("foreign key"), // standard sqlite foreign key error wraps/indicates failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			followRepo := NewFollowRepository(db)

			tt.setup(t, userRepo, followRepo)

			err := followRepo.CreateFollowRequest(tt.sender, tt.receiver)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if errors.Is(tt.wantErr, apperror.ErrConflict) && !errors.Is(err, apperror.ErrConflict) {
					t.Errorf("expected ErrConflict, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Verify request is pending
				fr, err := followRepo.GetFollowRequest(tt.sender, tt.receiver)
				if err != nil {
					t.Errorf("failed to get follow request: %v", err)
				}
				if fr.Status != "pending" {
					t.Errorf("expected status 'pending', got %q", fr.Status)
				}
			}
		})
	}
}

func TestGetFollowRequest(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository)
		sender   string
		receiver string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
				if err := followRepo.CreateFollowRequest("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed to create request: %v", err)
				}
			},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  nil,
		},
		{
			name:     "not_found",
			setup:    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			followRepo := NewFollowRepository(db)

			tt.setup(t, userRepo, followRepo)

			fr, err := followRepo.GetFollowRequest(tt.sender, tt.receiver)
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
				if fr.SenderID != tt.sender || fr.ReceiverID != tt.receiver {
					t.Errorf("got %+v, want sender=%s, receiver=%s", fr, tt.sender, tt.receiver)
				}
			}
		})
	}
}

func TestAcceptFollowRequest(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository)
		sender   string
		receiver string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
				if err := followRepo.CreateFollowRequest("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed to create request: %v", err)
				}
			},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  nil,
		},
		{
			name:     "not_found_no_request",
			setup:    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  apperror.ErrNotFound,
		},
		{
			name: "not_found_already_accepted",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
				if err := followRepo.CreateFollowRequest("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if err := followRepo.AcceptFollowRequest("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed to accept: %v", err)
				}
			},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			followRepo := NewFollowRepository(db)

			tt.setup(t, userRepo, followRepo)

			err := followRepo.AcceptFollowRequest(tt.sender, tt.receiver)
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
				// Verify status is accepted
				fr, err := followRepo.GetFollowRequest(tt.sender, tt.receiver)
				if err != nil {
					t.Errorf("failed to get request: %v", err)
				}
				if fr.Status != "accepted" {
					t.Errorf("expected status 'accepted', got %q", fr.Status)
				}
				// Verify follower relationship is created
				isFollowing, err := followRepo.IsFollowing(tt.sender, tt.receiver)
				if err != nil {
					t.Errorf("IsFollowing failed: %v", err)
				}
				if !isFollowing {
					t.Errorf("expected sender to be following receiver")
				}
			}
		})
	}
}

func TestDeclineFollowRequest(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository)
		sender   string
		receiver string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
				if err := followRepo.CreateFollowRequest("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed to create request: %v", err)
				}
			},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  nil,
		},
		{
			name:     "not_found",
			setup:    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {},
			sender:   "user-1",
			receiver: "user-2",
			wantErr:  apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			followRepo := NewFollowRepository(db)

			tt.setup(t, userRepo, followRepo)

			err := followRepo.DeclineFollowRequest(tt.sender, tt.receiver)
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
				// Verify request is deleted
				_, err := followRepo.GetFollowRequest(tt.sender, tt.receiver)
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected request to be deleted (ErrNotFound), got: %v", err)
				}
			}
		})
	}
}

func TestCreateFollower(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	followRepo := NewFollowRepository(db)

	createTestUser(t, userRepo, "user-1", "user1@example.com")
	createTestUser(t, userRepo, "user-2", "user2@example.com")

	// Create follower
	if err := followRepo.CreateFollower("user-1", "user-2"); err != nil {
		t.Fatalf("CreateFollower failed: %v", err)
	}

	// Verify
	isFollowing, err := followRepo.IsFollowing("user-1", "user-2")
	if err != nil || !isFollowing {
		t.Errorf("expected following relationship, err: %v, isFollowing: %v", err, isFollowing)
	}

	// Test Idempotency: calling again should not error
	if err := followRepo.CreateFollower("user-1", "user-2"); err != nil {
		t.Errorf("CreateFollower second call failed: %v", err)
	}
}

func TestDeleteFollower(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository)
		follower string
		followed string
		wantErr  error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")
				if err := followRepo.CreateFollower("user-1", "user-2"); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			follower: "user-1",
			followed: "user-2",
			wantErr:  nil,
		},
		{
			name:     "not_found",
			setup:    func(t *testing.T, userRepo UserRepository, followRepo FollowRepository) {},
			follower: "user-1",
			followed: "user-2",
			wantErr:  apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			followRepo := NewFollowRepository(db)

			tt.setup(t, userRepo, followRepo)

			err := followRepo.DeleteFollower(tt.follower, tt.followed)
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
				// Verify relationship is deleted
				isFollowing, err := followRepo.IsFollowing(tt.follower, tt.followed)
				if err != nil {
					t.Errorf("IsFollowing failed: %v", err)
				}
				if isFollowing {
					t.Errorf("expected relationship to be deleted")
				}
			}
		})
	}
}

func TestGetFollowersAndFollowing(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	followRepo := NewFollowRepository(db)

	// Create users: user-target, user-f1, user-f2
	createTestUser(t, userRepo, "target", "target@example.com")
	createTestUser(t, userRepo, "f1", "f1@example.com")
	createTestUser(t, userRepo, "f2", "f2@example.com")

	// Set up follower relationships: f1 follows target, f2 follows target
	if err := followRepo.CreateFollower("f1", "target"); err != nil {
		t.Fatalf("failed to create follower f1: %v", err)
	}
	if err := followRepo.CreateFollower("f2", "target"); err != nil {
		t.Fatalf("failed to create follower f2: %v", err)
	}

	// Update f1's created_at to be in the past so f2 is guaranteed to be newer when sorted by created_at DESC
	if _, err := db.Exec("UPDATE followers SET created_at = datetime('now', '-1 minute') WHERE follower_id = 'f1'"); err != nil {
		t.Fatalf("failed to update created_at for f1: %v", err)
	}

	// Set up following relationships: target follows f1
	if err := followRepo.CreateFollower("target", "f1"); err != nil {
		t.Fatalf("failed to create follower target: %v", err)
	}

	// 1. GetFollowers
	followers, err := followRepo.GetFollowers("target", 10, 0)
	if err != nil {
		t.Fatalf("GetFollowers failed: %v", err)
	}
	if len(followers) != 2 {
		t.Errorf("expected 2 followers, got %d", len(followers))
	}
	// Verify sorting (order by f.created_at DESC, so f2 should be first, then f1)
	if followers[0].ID != "f2" || followers[1].ID != "f1" {
		t.Errorf("expected followers order f2, f1; got %s, %s", followers[0].ID, followers[1].ID)
	}

	// Test offset & limit
	followersLimit, err := followRepo.GetFollowers("target", 1, 1)
	if err != nil {
		t.Fatalf("GetFollowers with limit failed: %v", err)
	}
	if len(followersLimit) != 1 || followersLimit[0].ID != "f1" {
		t.Errorf("expected follower f1 at offset 1, limit 1; got %+v", followersLimit)
	}

	// 2. GetFollowing
	following, err := followRepo.GetFollowing("target", 10, 0)
	if err != nil {
		t.Fatalf("GetFollowing failed: %v", err)
	}
	if len(following) != 1 || following[0].ID != "f1" {
		t.Errorf("expected target to follow f1, got %+v", following)
	}

	// 3. Counts
	followerCount, err := followRepo.GetFollowerCount("target")
	if err != nil || followerCount != 2 {
		t.Errorf("expected follower count 2, got %d, err: %v", followerCount, err)
	}

	followingCount, err := followRepo.GetFollowingCount("target")
	if err != nil || followingCount != 1 {
		t.Errorf("expected following count 1, got %d, err: %v", followingCount, err)
	}
}

func TestGetPendingRequests(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	followRepo := NewFollowRepository(db)

	createTestUser(t, userRepo, "user-1", "user1@example.com")
	createTestUser(t, userRepo, "user-2", "user2@example.com")
	createTestUser(t, userRepo, "user-3", "user3@example.com")

	// 1. pending request: user-1 -> user-2
	if err := followRepo.CreateFollowRequest("user-1", "user-2"); err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// 2. pending request: user-3 -> user-2
	if err := followRepo.CreateFollowRequest("user-3", "user-2"); err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Update user-1's request created_at to be in the past so user-3 is guaranteed to be newer when sorted by created_at DESC
	if _, err := db.Exec("UPDATE follow_requests SET created_at = datetime('now', '-1 minute') WHERE sender_id = 'user-1'"); err != nil {
		t.Fatalf("failed to update created_at for user-1 request: %v", err)
	}

	// Get pending requests for user-2
	reqs, err := followRepo.GetPendingRequests("user-2")
	if err != nil {
		t.Fatalf("GetPendingRequests failed: %v", err)
	}

	if len(reqs) != 2 {
		t.Errorf("expected 2 pending requests, got %d", len(reqs))
	}
	// Verify sorting (order by created_at DESC, so user-3 should be first, then user-1)
	if reqs[0].SenderID != "user-3" || reqs[1].SenderID != "user-1" {
		t.Errorf("expected order user-3, user-1; got %s, %s", reqs[0].SenderID, reqs[1].SenderID)
	}
}
