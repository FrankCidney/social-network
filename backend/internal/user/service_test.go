package user

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	// Restrict to 1 connection to prevent concurrency issues and match single-conn behavior.
	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		db.Close()
	})

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

type fakeMultipartFile struct {
	*bytes.Reader
}

func (f fakeMultipartFile) Close() error { return nil }

func TestGetProfile(t *testing.T) {
	const (
		viewerID = "viewer-123"
		targetID = "target-456"
	)

	tests := []struct {
		name      string
		viewerID  string
		targetID  string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		assertErr func(t *testing.T, err error)
		assertRes func(t *testing.T, profile *models.Profile)
	}{
		{
			name:     "Owner viewer (full profile)",
			viewerID: targetID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{
					ID:        targetID,
					Email:     "target@example.com",
					Password:  "hash",
					FirstName: "Target",
					LastName:  "User",
					DOB:       "1990-01-01",
					AboutMe:   "Secret bio",
					IsPublic:  false,
				})
				_, _ = db.Exec(
					`INSERT INTO posts (id, user_id, content, privacy, created_at)
					 VALUES ('post-1', ?, 'hello', 'public', datetime('now'))`,
					targetID,
				)
			},
			assertRes: func(t *testing.T, profile *models.Profile) {
				if profile.User.ID != targetID {
					t.Errorf("expected ID %q, got %q", targetID, profile.User.ID)
				}
				if !profile.IsOwnProfile {
					t.Error("owner profile should be marked as own profile")
				}
				if !profile.CanViewFullProfile {
					t.Error("owner should be allowed to view full profile")
				}
				if profile.Email != "target@example.com" {
					t.Errorf("owner should see email, got %q", profile.Email)
				}
				if profile.AboutMe != "Secret bio" || !strings.HasPrefix(profile.DOB, "1990-01-01") {
					t.Error("owner should see full profile info")
				}
				if profile.PostCount != 1 {
					t.Errorf("expected post count 1, got %d", profile.PostCount)
				}
			},
		},
		{
			name:     "Public profile (full profile to anyone)",
			viewerID: viewerID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{
					ID:        targetID,
					Email:     "target@example.com",
					Password:  "hash",
					FirstName: "Target",
					LastName:  "User",
					DOB:       "1990-01-01",
					AboutMe:   "Public bio",
					IsPublic:  true,
				})
			},
			assertRes: func(t *testing.T, profile *models.Profile) {
				if profile.IsOwnProfile {
					t.Error("public non-owner profile should not be marked as own profile")
				}
				if !profile.CanViewFullProfile {
					t.Error("public profile should be fully visible")
				}
				if profile.Email != "" {
					t.Errorf("non-owner should not see email, got %q", profile.Email)
				}
				if profile.AboutMe != "Public bio" || !strings.HasPrefix(profile.DOB, "1990-01-01") {
					t.Error("anyone should see full profile of public user")
				}
			},
		},
		{
			name:     "Private profile, confirmed follower (full profile)",
			viewerID: viewerID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{
					ID:    viewerID,
					Email: "viewer@example.com",
				})
				_ = users.CreateUser(&models.User{
					ID:        targetID,
					Email:     "target@example.com",
					Password:  "hash",
					FirstName: "Target",
					LastName:  "User",
					DOB:       "1990-01-01",
					AboutMe:   "Secret bio",
					IsPublic:  false,
				})
				_ = follows.CreateFollower(viewerID, targetID)
			},
			assertRes: func(t *testing.T, profile *models.Profile) {
				if !profile.IsFollowing {
					t.Error("confirmed follower should be marked as following")
				}
				if !profile.CanViewFullProfile {
					t.Error("confirmed follower should be allowed to view full profile")
				}
				if profile.Email != "" {
					t.Errorf("follower should not see email, got %q", profile.Email)
				}
				if profile.AboutMe != "Secret bio" || !strings.HasPrefix(profile.DOB, "1990-01-01") {
					t.Error("confirmed follower should see full profile")
				}
			},
		},
		{
			name:     "Private profile, non-follower (restricted profile)",
			viewerID: viewerID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{
					ID:        targetID,
					Email:     "target@example.com",
					Password:  "hash",
					FirstName: "Target",
					LastName:  "User",
					DOB:       "1990-01-01",
					AboutMe:   "Secret bio",
					IsPublic:  false,
				})
			},
			assertRes: func(t *testing.T, profile *models.Profile) {
				if profile.AboutMe != "" || profile.DOB != "" {
					t.Error("non-follower should NOT see about me or dob on private profile")
				}
				if profile.CanViewFullProfile {
					t.Error("non-follower should not be allowed to view full profile")
				}
				if profile.IsFollowing {
					t.Error("non-follower should not be marked as following")
				}
				if profile.FollowerCount != 0 || profile.FollowingCount != 0 || profile.PostCount != 0 {
					t.Errorf("restricted private profile should not expose counts: %+v", profile)
				}
				if profile.Email != "" {
					t.Errorf("non-owner should not see email, got %q", profile.Email)
				}
			},
		},
		{
			name:     "Private profile with pending follow request",
			viewerID: viewerID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{
					ID:    viewerID,
					Email: "viewer@example.com",
				})
				_ = users.CreateUser(&models.User{
					ID:        targetID,
					Email:     "target@example.com",
					Password:  "hash",
					FirstName: "Target",
					LastName:  "User",
					DOB:       "1990-01-01",
					AboutMe:   "Secret bio",
					IsPublic:  false,
				})
				_ = follows.CreateFollowRequest(viewerID, targetID)
			},
			assertRes: func(t *testing.T, profile *models.Profile) {
				if profile.FollowRequestStatus != "pending" {
					t.Errorf("expected pending follow request, got %q", profile.FollowRequestStatus)
				}
				if profile.CanViewFullProfile {
					t.Error("pending requester should not be allowed to view full profile")
				}
				if profile.IsFollowing {
					t.Error("pending request should not be marked as following")
				}
				if profile.AboutMe != "" || profile.DOB != "" {
					t.Error("pending requester should not see private profile details")
				}
			},
		},
		{
			name:     "Target user not found",
			viewerID: viewerID,
			targetID: "nonexistent",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Database error on follows count",
			viewerID: viewerID,
			targetID: targetID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{
					ID:    targetID,
					Email: "target@example.com",
				})
				_, _ = db.Exec("DROP TABLE followers")
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			follows := repository.NewFollowRepository(db)
			posts := repository.NewPostRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, follows)
			}

			svc := NewService(users, follows, posts)
			res, err := svc.GetProfile(tt.viewerID, tt.targetID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if res != nil {
					t.Errorf("expected nil profile, got %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res == nil {
					t.Fatal("expected non-nil profile")
				}
				if tt.assertRes != nil {
					tt.assertRes(t, res)
				}
			}
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	const userID = "user-123"

	tests := []struct {
		name      string
		req       models.UpdateProfileRequest
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository)
		assertErr func(t *testing.T, err error)
		assertDB  func(t *testing.T, users repository.UserRepository)
	}{
		{
			name: "Success partial update",
			req: models.UpdateProfileRequest{
				FirstName: "UpdatedFirst",
				LastName:  "UpdatedLast",
				AboutMe:   "New bio",
			},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_ = users.CreateUser(&models.User{
					ID:        userID,
					Email:     "user@example.com",
					FirstName: "OldFirst",
					LastName:  "OldLast",
					DOB:       "1990-01-01",
					AboutMe:   "Old bio",
				})
			},
			assertDB: func(t *testing.T, users repository.UserRepository) {
				u, err := users.GetUserByID(userID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if u.FirstName != "UpdatedFirst" || u.LastName != "UpdatedLast" || u.AboutMe != "New bio" {
					t.Errorf("unexpected user state: %+v", u)
				}
				if !strings.HasPrefix(u.DOB, "1990-01-01") {
					t.Errorf("DOB should remain unchanged, got %q", u.DOB)
				}
			},
		},
		{
			name: "Success update DOB and visibility",
			req: func() models.UpdateProfileRequest {
				isPublic := false
				return models.UpdateProfileRequest{
					DOB:      "2000-12-31",
					IsPublic: &isPublic,
				}
			}(),
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_ = users.CreateUser(&models.User{
					ID:       userID,
					Email:    "user@example.com",
					DOB:      "1990-01-01",
					IsPublic: true,
				})
			},
			assertDB: func(t *testing.T, users repository.UserRepository) {
				u, err := users.GetUserByID(userID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if !strings.HasPrefix(u.DOB, "2000-12-31") || u.IsPublic != false {
					t.Errorf("unexpected user state: %+v", u)
				}
			},
		},
		{
			name: "Invalid DOB format",
			req: models.UpdateProfileRequest{
				DOB: "31-12-2000",
			},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_ = users.CreateUser(&models.User{
					ID:    userID,
					Email: "user@example.com",
				})
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "User not found",
			req: models.UpdateProfileRequest{
				FirstName: "New",
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "Database error on update",
			req: models.UpdateProfileRequest{
				FirstName: "New",
			},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_ = users.CreateUser(&models.User{
					ID:    userID,
					Email: "user@example.com",
				})
				_, _ = db.Exec("DROP TABLE users")
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
				tt.setupDB(t, db, users)
			}

			svc := NewService(users, follows)
			err := svc.UpdateProfile(userID, tt.req)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.assertDB != nil {
					tt.assertDB(t, users)
				}
			}
		})
	}
}

func TestSetProfileVisibility(t *testing.T) {
	const userID = "user-123"

	tests := []struct {
		name      string
		isPublic  bool
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository)
		assertErr func(t *testing.T, err error)
		assertDB  func(t *testing.T, users repository.UserRepository)
	}{
		{
			name:     "Success set to private",
			isPublic: false,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_ = users.CreateUser(&models.User{
					ID:       userID,
					Email:    "user@example.com",
					IsPublic: true,
				})
			},
			assertDB: func(t *testing.T, users repository.UserRepository) {
				u, err := users.GetUserByID(userID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if u.IsPublic != false {
					t.Error("expected user to be private")
				}
			},
		},
		{
			name:     "User not found",
			isPublic: false,
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
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
				tt.setupDB(t, db, users)
			}

			svc := NewService(users, follows)
			err := svc.SetProfileVisibility(userID, tt.isPublic)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.assertDB != nil {
					tt.assertDB(t, users)
				}
			}
		})
	}
}

func TestUploadAvatar(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll("uploads")
	})

	const userID = "user-123"

	validPngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	tests := []struct {
		name      string
		fileBytes []byte
		filename  string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository)
		assertErr func(t *testing.T, err error)
		assertRes func(t *testing.T, path string, users repository.UserRepository)
	}{
		{
			name:      "Success upload png",
			fileBytes: validPngBytes,
			filename:  "avatar.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_ = users.CreateUser(&models.User{
					ID:    userID,
					Email: "user@example.com",
				})
			},
			assertRes: func(t *testing.T, path string, users repository.UserRepository) {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("expected file %q to be created, but it does not exist", path)
				}
				u, err := users.GetUserByID(userID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if u.AvatarPath != path {
					t.Errorf("expected DB avatar path to be %q, got %q", path, u.AvatarPath)
				}
			},
		},
		{
			name:      "Too large file",
			fileBytes: make([]byte, 6<<20), // 6 MB
			filename:  "avatar.png",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Invalid extension",
			fileBytes: validPngBytes,
			filename:  "avatar.txt",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Invalid image bytes",
			fileBytes: []byte("not-a-png-or-jpeg-or-gif"),
			filename:  "avatar.png",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "User not found",
			fileBytes: validPngBytes,
			filename:  "avatar.png",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
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
				tt.setupDB(t, db, users)
			}

			file := fakeMultipartFile{Reader: bytes.NewReader(tt.fileBytes)}
			header := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     int64(len(tt.fileBytes)),
			}

			svc := NewService(users, follows)
			path, err := svc.UploadAvatar(userID, file, header)

			if path != "" {
				t.Cleanup(func() {
					_ = os.Remove(path)
				})
			}

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if path != "" {
					t.Errorf("expected empty path on error, got %q", path)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if path == "" {
					t.Fatal("expected non-empty path")
				}
				if tt.assertRes != nil {
					tt.assertRes(t, path, users)
				}
			}
		})
	}
}

func TestGetFollowersAndFollowing(t *testing.T) {
	const userID = "user-123"

	tests := []struct {
		name      string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository)
		testFunc  func(t *testing.T, svc Service)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "Success GetFollowers and GetFollowing",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: userID, Email: "user@example.com"})
				_ = users.CreateUser(&models.User{ID: "f1", Email: "f1@example.com"})
				_ = users.CreateUser(&models.User{ID: "f2", Email: "f2@example.com"})

				// f1 follows userID (userID has follower f1)
				_ = follows.CreateFollower("f1", userID)

				// userID follows f2 (userID is following f2)
				_ = follows.CreateFollower(userID, "f2")
			},
			testFunc: func(t *testing.T, svc Service) {
				followers, err := svc.GetFollowers(userID, userID, 10, 0)
				if err != nil {
					t.Fatalf("GetFollowers failed: %v", err)
				}
				if followers.Total != 1 || len(followers.Users) != 1 || followers.Users[0].ID != "f1" {
					t.Errorf("unexpected followers list: %+v", followers)
				}

				following, err := svc.GetFollowing(userID, userID, 10, 0)
				if err != nil {
					t.Fatalf("GetFollowing failed: %v", err)
				}
				if following.Total != 1 || len(following.Users) != 1 || following.Users[0].ID != "f2" {
					t.Errorf("unexpected following list: %+v", following)
				}
			},
		},
		{
			name: "User not found GetFollowers",
			testFunc: func(t *testing.T, svc Service) {
				_, err := svc.GetFollowers("viewer", "nonexistent", 10, 0)
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "User not found GetFollowing",
			testFunc: func(t *testing.T, svc Service) {
				_, err := svc.GetFollowing("viewer", "nonexistent", 10, 0)
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "Private profile hides follower and following lists from non-followers",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: "viewer", Email: "viewer@example.com"})
				_ = users.CreateUser(&models.User{ID: userID, Email: "user@example.com", IsPublic: false})
				_ = users.CreateUser(&models.User{ID: "f1", Email: "f1@example.com"})
				_ = users.CreateUser(&models.User{ID: "f2", Email: "f2@example.com"})
				_ = follows.CreateFollower("f1", userID)
				_ = follows.CreateFollower(userID, "f2")
			},
			testFunc: func(t *testing.T, svc Service) {
				_, err := svc.GetFollowers("viewer", userID, 10, 0)
				if !errors.Is(err, apperror.ErrForbidden) {
					t.Fatalf("expected forbidden on followers, got %v", err)
				}

				_, err = svc.GetFollowing("viewer", userID, 10, 0)
				if !errors.Is(err, apperror.ErrForbidden) {
					t.Fatalf("expected forbidden on following, got %v", err)
				}
			},
		},
		{
			name: "Database error on GetFollowers list",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, follows repository.FollowRepository) {
				_ = users.CreateUser(&models.User{ID: userID, Email: "user@example.com"})
				_, _ = db.Exec("DROP TABLE followers")
			},
			testFunc: func(t *testing.T, svc Service) {
				_, err := svc.GetFollowers(userID, userID, 10, 0)
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

			svc := NewService(users, follows)
			tt.testFunc(t, svc)
		})
	}
}
