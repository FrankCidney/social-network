package post

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

type mockGroupMembership struct {
	isMemberFunc func(groupID, userID string) (bool, error)
}

func (m *mockGroupMembership) IsMember(groupID, userID string) (bool, error) {
	if m.isMemberFunc != nil {
		return m.isMemberFunc(groupID, userID)
	}
	return false, nil
}

type mockFile struct {
	*bytes.Reader
}

func (m *mockFile) Close() error {
	return nil
}

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

func seedUser(t *testing.T, db *sql.DB, id, email string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (id, email, password, first_name, last_name, dob, nickname, about_me, is_public, created_at)
		VALUES (?, ?, 'hash', 'First', 'Last', '1990-01-01', 'nick', 'about', 1, datetime('now'))`,
		id, email,
	)
	if err != nil {
		t.Fatalf("failed to seed user %s: %v", id, err)
	}
}

func seedGroup(t *testing.T, db *sql.DB, id, creatorID, title string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES (?, ?, ?, 'description', datetime('now'))`,
		id, creatorID, title,
	)
	if err != nil {
		t.Fatalf("failed to seed group %s: %v", id, err)
	}
}

func seedGroupMember(t *testing.T, db *sql.DB, groupID, userID, status string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO group_members (group_id, user_id, status, created_at)
		VALUES (?, ?, ?, datetime('now'))`,
		groupID, userID, status,
	)
	if err != nil {
		t.Fatalf("failed to seed group member %s/%s: %v", groupID, userID, err)
	}
}

func seedFollow(t *testing.T, db *sql.DB, followerID, followedID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO followers (follower_id, followed_id, created_at)
		VALUES (?, ?, datetime('now'))`,
		followerID, followedID,
	)
	if err != nil {
		t.Fatalf("failed to seed follower %s -> %s: %v", followerID, followedID, err)
	}
}

func seedPost(t *testing.T, db *sql.DB, id, userID string, groupID *string, content, privacy string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO posts (id, user_id, group_id, content, privacy, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		id, userID, groupID, content, privacy,
	)
	if err != nil {
		t.Fatalf("failed to seed post %s: %v", id, err)
	}
}

func seedPostVisibility(t *testing.T, db *sql.DB, postID, userID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO post_visibility (post_id, user_id)
		VALUES (?, ?)`,
		postID, userID,
	)
	if err != nil {
		t.Fatalf("failed to seed post visibility %s/%s: %v", postID, userID, err)
	}
}

func seedTestUsersAndFollowers(t *testing.T, db *sql.DB) {
	t.Helper()
	seedUser(t, db, "user-1", "user1@example.com")
	seedUser(t, db, "user-2", "user2@example.com")
	seedUser(t, db, "user-3", "user3@example.com")
	seedFollow(t, db, "user-2", "user-1") // user-2 follows user-1
}

func seedTestFeedData(t *testing.T, db *sql.DB) {
	t.Helper()
	// Seed users
	seedUser(t, db, "user-1", "user1@example.com")
	seedUser(t, db, "user-2", "user2@example.com")
	seedUser(t, db, "user-3", "user3@example.com")

	// user-1 follows user-2
	seedFollow(t, db, "user-1", "user-2")

	// Seed groups
	seedGroup(t, db, "group-1", "user-3", "Group 1")
	seedGroup(t, db, "group-2", "user-3", "Group 2")

	// seed group membership
	seedGroupMember(t, db, "group-1", "user-1", "accepted")
	seedGroupMember(t, db, "group-2", "user-1", "invited") // not accepted, so not a member!

	// Seed posts
	gID1 := "group-1"
	gID2 := "group-2"

	// Insert posts with explicit custom timestamps in RFC3339 format to ensure deterministic ordering.
	insertPostWithTime := func(id, userID string, groupID *string, content, privacy, createdAt string) {
		_, err := db.Exec(`
			INSERT INTO posts (id, user_id, group_id, content, privacy, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, userID, groupID, content, privacy, createdAt,
		)
		if err != nil {
			t.Fatalf("failed to insert post %s: %v", id, err)
		}
	}

	insertPostWithTime("post-1", "user-2", nil, "Public User 2", "public", "2026-06-30T10:00:00Z")
	insertPostWithTime("post-2", "user-2", nil, "Almost Private User 2", "almost_private", "2026-06-30T10:01:00Z")
	insertPostWithTime("post-3", "user-2", nil, "Private User 2 Not Visible", "private", "2026-06-30T10:02:00Z")
	insertPostWithTime("post-4", "user-2", nil, "Private User 2 Visible", "private", "2026-06-30T10:03:00Z")
	seedPostVisibility(t, db, "post-4", "user-1")

	insertPostWithTime("post-5", "user-3", nil, "Public User 3", "public", "2026-06-30T10:04:00Z")
	insertPostWithTime("post-6", "user-3", nil, "Almost Private User 3", "almost_private", "2026-06-30T10:05:00Z")
	insertPostWithTime("post-7", "user-3", &gID1, "Group 1 Post", "group", "2026-06-30T10:06:00Z")
	insertPostWithTime("post-8", "user-3", &gID2, "Group 2 Post", "group", "2026-06-30T10:07:00Z")
}

func TestCreatePost(t *testing.T) {
	gID1 := "group-1"
	gID2 := "group-2"
	emptyGroupID := ""

	tests := []struct {
		name         string
		authorID     string
		req          models.CreatePostRequest
		setupDB      func(t *testing.T, db *sql.DB)
		mockGroups   *mockGroupMembership
		assertErr    func(t *testing.T, err error)
		expectResult func(t *testing.T, p *models.Post)
	}{
		{
			name:     "Success Public Post",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello world",
				Privacy: "public",
			},
			expectResult: func(t *testing.T, p *models.Post) {
				if p.Content != "Hello world" {
					t.Errorf("expected Content %q, got %q", "Hello world", p.Content)
				}
				if p.Privacy != "public" {
					t.Errorf("expected Privacy %q, got %q", "public", p.Privacy)
				}
				if p.GroupID != nil {
					t.Errorf("expected nil GroupID, got %v", p.GroupID)
				}
			},
		},
		{
			name:     "Success Group Post",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello group",
				Privacy: "group",
				GroupID: &gID1,
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedGroup(t, db, "group-1", "user-1", "My Group")
			},
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					if groupID == "group-1" && userID == "user-1" {
						return true, nil
					}
					return false, nil
				},
			},
			expectResult: func(t *testing.T, p *models.Post) {
				if p.Content != "Hello group" {
					t.Errorf("expected Content %q, got %q", "Hello group", p.Content)
				}
				if p.Privacy != "group" {
					t.Errorf("expected Privacy %q, got %q", "group", p.Privacy)
				}
				if p.GroupID == nil || *p.GroupID != "group-1" {
					t.Errorf("expected GroupID %q, got %v", "group-1", p.GroupID)
				}
			},
		},
		{
			name:     "Success Private Post with Follower",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content:   "Hello private",
				Privacy:   "private",
				VisibleTo: []string{"user-2"},
			},
			expectResult: func(t *testing.T, p *models.Post) {
				if p.Content != "Hello private" {
					t.Errorf("expected Content %q, got %q", "Hello private", p.Content)
				}
				if p.Privacy != "private" {
					t.Errorf("expected Privacy %q, got %q", "private", p.Privacy)
				}
			},
		},
		{
			name:     "Failure Private Post with Non-Follower",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content:   "Hello private",
				Privacy:   "private",
				VisibleTo: []string{"user-3"},
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Failure Group Post when Not Member",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello group",
				Privacy: "group",
				GroupID: &gID2,
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedGroup(t, db, "group-2", "user-2", "Other Group")
			},
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					return false, nil
				},
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrForbidden) {
					t.Errorf("expected ErrForbidden, got %v", err)
				}
			},
		},
		{
			name:     "Failure Group Post with VisibleTo set",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content:   "Hello group",
				Privacy:   "group",
				GroupID:   &gID1,
				VisibleTo: []string{"user-2"},
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedGroup(t, db, "group-1", "user-1", "My Group")
			},
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					return true, nil
				},
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Failure Group Post with GroupID nil",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello group",
				Privacy: "group",
				GroupID: nil,
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Failure Group Post with GroupID empty",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello group",
				Privacy: "group",
				GroupID: &emptyGroupID,
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Failure Non-Group Post with GroupID set",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello public",
				Privacy: "public",
				GroupID: &gID1,
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Failure Invalid Privacy",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello",
				Privacy: "super-private",
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Failure Database Error",
			authorID: "user-1",
			req: models.CreatePostRequest{
				Content: "Hello",
				Privacy: "public",
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				_, err := db.Exec("DROP TABLE posts")
				if err != nil {
					t.Fatalf("failed to drop posts table: %v", err)
				}
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
			seedTestUsersAndFollowers(t, db)

			if tt.setupDB != nil {
				tt.setupDB(t, db)
			}

			postsRepo := repository.NewPostRepository(db)
			usersRepo := repository.NewUserRepository(db)
			followsRepo := repository.NewFollowRepository(db)

			var groups GroupMembership = &mockGroupMembership{}
			if tt.mockGroups != nil {
				groups = tt.mockGroups
			}

			svc := NewService(postsRepo, usersRepo, followsRepo, groups)
			res, err := svc.CreatePost(tt.authorID, tt.req)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if res != nil {
					t.Errorf("expected nil response, got %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.expectResult != nil {
					tt.expectResult(t, res)
				}
			}
		})
	}
}

func TestGetPost(t *testing.T) {
	gID1 := "group-1"

	setupPosts := func(t *testing.T, db *sql.DB) {
		seedPost(t, db, "post-public", "user-1", nil, "Public Content", "public")
		seedPost(t, db, "post-almost", "user-1", nil, "Almost Private Content", "almost_private")
		seedPost(t, db, "post-private", "user-1", nil, "Private Content", "private")
		seedPostVisibility(t, db, "post-private", "user-2")

		seedGroup(t, db, "group-1", "user-1", "Group 1")
		seedPost(t, db, "post-group", "user-1", &gID1, "Group Content", "group")
	}

	tests := []struct {
		name         string
		viewerID     string
		postID       string
		setupDB      func(t *testing.T, db *sql.DB)
		mockGroups   *mockGroupMembership
		assertErr    func(t *testing.T, err error)
		expectResult func(t *testing.T, res *models.PostResponse)
	}{
		{
			name:     "Owner View Public",
			viewerID: "user-1",
			postID:   "post-public",
			expectResult: func(t *testing.T, res *models.PostResponse) {
				if res.ID != "post-public" {
					t.Errorf("expected post-public, got %s", res.ID)
				}
				if res.Author.ID != "user-1" {
					t.Errorf("expected author user-1, got %s", res.Author.ID)
				}
			},
		},
		{
			name:     "Non-Owner View Public",
			viewerID: "user-3",
			postID:   "post-public",
			expectResult: func(t *testing.T, res *models.PostResponse) {
				if res.ID != "post-public" {
					t.Errorf("expected post-public, got %s", res.ID)
				}
			},
		},
		{
			name:     "Owner View Almost Private",
			viewerID: "user-1",
			postID:   "post-almost",
			expectResult: func(t *testing.T, res *models.PostResponse) {
				if res.ID != "post-almost" {
					t.Errorf("expected post-almost, got %s", res.ID)
				}
			},
		},
		{
			name:     "Follower View Almost Private",
			viewerID: "user-2",
			postID:   "post-almost",
			expectResult: func(t *testing.T, res *models.PostResponse) {
				if res.ID != "post-almost" {
					t.Errorf("expected post-almost, got %s", res.ID)
				}
			},
		},
		{
			name:     "Non-Follower View Almost Private",
			viewerID: "user-3",
			postID:   "post-almost",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Allowed Viewer View Private",
			viewerID: "user-2",
			postID:   "post-private",
			expectResult: func(t *testing.T, res *models.PostResponse) {
				if res.ID != "post-private" {
					t.Errorf("expected post-private, got %s", res.ID)
				}
			},
		},
		{
			name:     "Non-Allowed Viewer View Private",
			viewerID: "user-3",
			postID:   "post-private",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Group Member View Group Post",
			viewerID: "user-2",
			postID:   "post-group",
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					if groupID == "group-1" && userID == "user-2" {
						return true, nil
					}
					return false, nil
				},
			},
			expectResult: func(t *testing.T, res *models.PostResponse) {
				if res.ID != "post-group" {
					t.Errorf("expected post-group, got %s", res.ID)
				}
			},
		},
		{
			name:     "Non-Group Member View Group Post",
			viewerID: "user-3",
			postID:   "post-group",
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					return false, nil
				},
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Post Not Found",
			viewerID: "user-1",
			postID:   "nonexistent",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Author Not Found DB Error",
			viewerID: "user-2",
			postID:   "post-public",
			setupDB: func(t *testing.T, db *sql.DB) {
				_, err := db.Exec("PRAGMA foreign_keys = OFF")
				if err != nil {
					t.Fatalf("disable foreign keys: %v", err)
				}
				_, err = db.Exec("DELETE FROM users WHERE id = 'user-1'")
				if err != nil {
					t.Fatalf("delete user: %v", err)
				}
				_, err = db.Exec("PRAGMA foreign_keys = ON")
				if err != nil {
					t.Fatalf("enable foreign keys: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error when author is deleted, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			seedTestUsersAndFollowers(t, db)
			setupPosts(t, db)

			if tt.setupDB != nil {
				tt.setupDB(t, db)
			}

			postsRepo := repository.NewPostRepository(db)
			usersRepo := repository.NewUserRepository(db)
			followsRepo := repository.NewFollowRepository(db)

			var groups GroupMembership = &mockGroupMembership{}
			if tt.mockGroups != nil {
				groups = tt.mockGroups
			}

			svc := NewService(postsRepo, usersRepo, followsRepo, groups)
			res, err := svc.GetPost(tt.viewerID, tt.postID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if res != nil {
					t.Errorf("expected nil response, got %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.expectResult != nil {
					tt.expectResult(t, res)
				}
			}
		})
	}
}

func TestUpdatePost(t *testing.T) {
	tests := []struct {
		name      string
		authorID  string
		postID    string
		req       models.UpdatePostRequest
		setupDB   func(t *testing.T, db *sql.DB)
		assertErr func(t *testing.T, err error)
		verify    func(t *testing.T, db *sql.DB, posts repository.PostRepository)
	}{
		{
			name:     "Success Update Content and Privacy",
			authorID: "user-1",
			postID:   "post-1",
			req: models.UpdatePostRequest{
				Content: "Updated content",
				Privacy: "public",
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Old Content", "private")
				seedPostVisibility(t, db, "post-1", "user-2")
			},
			verify: func(t *testing.T, db *sql.DB, posts repository.PostRepository) {
				p, err := posts.GetPostByID("post-1")
				if err != nil {
					t.Fatalf("get post: %v", err)
				}
				if p.Content != "Updated content" {
					t.Errorf("expected 'Updated content', got %q", p.Content)
				}
				if p.Privacy != "public" {
					t.Errorf("expected 'public', got %q", p.Privacy)
				}
				// Verify visibility is cleared
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM post_visibility WHERE post_id = 'post-1'").Scan(&count)
				if err != nil {
					t.Fatalf("query visibility count: %v", err)
				}
				if count != 0 {
					t.Errorf("expected visibility count 0, got %d", count)
				}
			},
		},
		{
			name:     "Success Change to Private",
			authorID: "user-1",
			postID:   "post-1",
			req: models.UpdatePostRequest{
				Content:   "New private post",
				Privacy:   "private",
				VisibleTo: []string{"user-2"},
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Public Content", "public")
			},
			verify: func(t *testing.T, db *sql.DB, posts repository.PostRepository) {
				p, err := posts.GetPostByID("post-1")
				if err != nil {
					t.Fatalf("get post: %v", err)
				}
				if p.Privacy != "private" {
					t.Errorf("expected private, got %q", p.Privacy)
				}
				var exists bool
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM post_visibility WHERE post_id = 'post-1' AND user_id = 'user-2')").Scan(&exists)
				if err != nil {
					t.Fatalf("query visibility: %v", err)
				}
				if !exists {
					t.Error("expected user-2 to be visible_to")
				}
			},
		},
		{
			name:     "Failure Non-Owner",
			authorID: "user-2",
			postID:   "post-1",
			req: models.UpdatePostRequest{
				Content: "Hack",
				Privacy: "public",
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Failure Post Not Found",
			authorID: "user-1",
			postID:   "nonexistent",
			req: models.UpdatePostRequest{
				Content: "Hack",
				Privacy: "public",
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Failure Privacy Invariant Violation",
			authorID: "user-1",
			postID:   "post-1",
			req: models.UpdatePostRequest{
				Content:   "Content",
				Privacy:   "private",
				VisibleTo: []string{"user-3"}, // user-3 is not a follower
			},
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			seedTestUsersAndFollowers(t, db)

			if tt.setupDB != nil {
				tt.setupDB(t, db)
			}

			postsRepo := repository.NewPostRepository(db)
			usersRepo := repository.NewUserRepository(db)
			followsRepo := repository.NewFollowRepository(db)

			svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})
			err := svc.UpdatePost(tt.authorID, tt.postID, tt.req)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.verify != nil {
					tt.verify(t, db, postsRepo)
				}
			}
		})
	}
}

func TestDeletePost(t *testing.T) {
	tests := []struct {
		name      string
		authorID  string
		postID    string
		setupDB   func(t *testing.T, db *sql.DB)
		assertErr func(t *testing.T, err error)
		verify    func(t *testing.T, db *sql.DB)
	}{
		{
			name:     "Success Owner Delete",
			authorID: "user-1",
			postID:   "post-1",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			verify: func(t *testing.T, db *sql.DB) {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = 'post-1'").Scan(&count)
				if err != nil {
					t.Fatalf("query count: %v", err)
				}
				if count != 0 {
					t.Errorf("expected post to be deleted, got count %d", count)
				}
			},
		},
		{
			name:     "Failure Non-Owner Delete",
			authorID: "user-2",
			postID:   "post-1",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Failure Post Not Found",
			authorID: "user-1",
			postID:   "nonexistent",
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
			seedTestUsersAndFollowers(t, db)

			if tt.setupDB != nil {
				tt.setupDB(t, db)
			}

			postsRepo := repository.NewPostRepository(db)
			usersRepo := repository.NewUserRepository(db)
			followsRepo := repository.NewFollowRepository(db)

			svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})
			err := svc.DeletePost(tt.authorID, tt.postID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.verify != nil {
					tt.verify(t, db)
				}
			}
		})
	}
}

func TestUploadPostImage(t *testing.T) {
	validJPEGBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0, 0, 0, 0}

	tests := []struct {
		name      string
		authorID  string
		postID    string
		fileBytes []byte
		filename  string
		setupDB   func(t *testing.T, db *sql.DB)
		assertErr func(t *testing.T, err error)
		verify    func(t *testing.T, db *sql.DB, posts repository.PostRepository, path string)
	}{
		{
			name:      "Success Upload",
			authorID:  "user-1",
			postID:    "post-1",
			fileBytes: validJPEGBytes,
			filename:  "image.jpg",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			verify: func(t *testing.T, db *sql.DB, posts repository.PostRepository, path string) {
				// Verify path starts with uploads/posts/
				if !strings.HasPrefix(path, "uploads/posts/") {
					t.Errorf("expected path to start with uploads/posts/, got %q", path)
				}
				// Verify file exists on disk
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("expected file %q to exist on disk", path)
				}
				// Verify post ImageURL is updated in db
				p, err := posts.GetPostByID("post-1")
				if err != nil {
					t.Fatalf("get post: %v", err)
				}
				if p.ImageURL != path {
					t.Errorf("expected post ImageURL %q, got %q", path, p.ImageURL)
				}
			},
		},
		{
			name:      "Failure Non-Owner",
			authorID:  "user-2",
			postID:    "post-1",
			fileBytes: validJPEGBytes,
			filename:  "image.jpg",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:      "Failure Post Not Found",
			authorID:  "user-1",
			postID:    "nonexistent",
			fileBytes: validJPEGBytes,
			filename:  "image.jpg",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:      "Failure Too Large Image",
			authorID:  "user-1",
			postID:    "post-1",
			fileBytes: make([]byte, 6<<20), // 6 MB
			filename:  "image.jpg",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Failure Invalid Extension",
			authorID:  "user-1",
			postID:    "post-1",
			fileBytes: validJPEGBytes,
			filename:  "image.exe",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Failure Invalid Magic Bytes",
			authorID:  "user-1",
			postID:    "post-1",
			fileBytes: []byte("not an image at all"),
			filename:  "image.jpg",
			setupDB: func(t *testing.T, db *sql.DB) {
				seedPost(t, db, "post-1", "user-1", nil, "Content", "public")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.RemoveAll("uploads")

			db := setupTestDB(t)
			seedTestUsersAndFollowers(t, db)

			if tt.setupDB != nil {
				tt.setupDB(t, db)
			}

			postsRepo := repository.NewPostRepository(db)
			usersRepo := repository.NewUserRepository(db)
			followsRepo := repository.NewFollowRepository(db)

			svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})

			fileReader := bytes.NewReader(tt.fileBytes)
			file := &mockFile{Reader: fileReader}
			header := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     int64(len(tt.fileBytes)),
			}

			path, err := svc.UploadPostImage(tt.authorID, tt.postID, file, header)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if path != "" {
					t.Errorf("expected empty path, got %q", path)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.verify != nil {
					tt.verify(t, db, postsRepo, path)
				}
			}
		})
	}
}

func TestGetFeed(t *testing.T) {
	db := setupTestDB(t)
	seedTestFeedData(t, db)

	postsRepo := repository.NewPostRepository(db)
	usersRepo := repository.NewUserRepository(db)
	followsRepo := repository.NewFollowRepository(db)

	svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})

	// Get feed for user-1
	res, err := svc.GetFeed("user-1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Total != 5 {
		t.Errorf("expected Total 5, got %d", res.Total)
	}

	expectedIDs := []string{"post-7", "post-5", "post-4", "post-2", "post-1"}
	if len(res.Posts) != len(expectedIDs) {
		t.Fatalf("expected %d posts, got %d", len(expectedIDs), len(res.Posts))
	}

	for i, expectedID := range expectedIDs {
		if res.Posts[i].ID != expectedID {
			t.Errorf("at index %d: expected post ID %q, got %q", i, expectedID, res.Posts[i].ID)
		}
	}

	// Test pagination limits
	resPaginated, err := svc.GetFeed("user-1", 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// offset 1, limit 2: should get "post-5", "post-4"
	expectedPaginatedIDs := []string{"post-5", "post-4"}
	if len(resPaginated.Posts) != len(expectedPaginatedIDs) {
		t.Fatalf("expected %d posts, got %d", len(expectedPaginatedIDs), len(resPaginated.Posts))
	}
	for i, expectedID := range expectedPaginatedIDs {
		if resPaginated.Posts[i].ID != expectedID {
			t.Errorf("at index %d: expected post ID %q, got %q", i, expectedID, resPaginated.Posts[i].ID)
		}
	}
}

func TestGetPostsByAuthor(t *testing.T) {
	db := setupTestDB(t)
	seedTestFeedData(t, db)

	postsRepo := repository.NewPostRepository(db)
	usersRepo := repository.NewUserRepository(db)
	followsRepo := repository.NewFollowRepository(db)

	svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})

	// user-1 views user-2's posts
	res, err := svc.GetPostsByAuthor("user-1", "user-2", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Total != 4 {
		t.Errorf("expected Total 4, got %d", res.Total)
	}

	expectedIDs := []string{"post-4", "post-2", "post-1"}
	if len(res.Posts) != len(expectedIDs) {
		t.Fatalf("expected %d posts, got %d", len(expectedIDs), len(res.Posts))
	}

	for i, expectedID := range expectedIDs {
		if res.Posts[i].ID != expectedID {
			t.Errorf("at index %d: expected post ID %q, got %q", i, expectedID, res.Posts[i].ID)
		}
	}
}

func TestGetPostsByAuthorRejectsPrivateProfileForNonFollower(t *testing.T) {
	db := setupTestDB(t)
	seedUser(t, db, "viewer", "viewer@example.com")
	seedUser(t, db, "author", "author@example.com")

	if _, err := db.Exec(`UPDATE users SET is_public = 0 WHERE id = 'author'`); err != nil {
		t.Fatalf("failed to make author private: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO posts (id, user_id, content, privacy, created_at)
		VALUES ('post-private-profile', 'author', 'hidden', 'public', datetime('now'))
	`); err != nil {
		t.Fatalf("failed to seed post: %v", err)
	}

	postsRepo := repository.NewPostRepository(db)
	usersRepo := repository.NewUserRepository(db)
	followsRepo := repository.NewFollowRepository(db)

	svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})

	_, err := svc.GetPostsByAuthor("viewer", "author", 10, 0)
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestGetGroupPosts(t *testing.T) {
	db := setupTestDB(t)
	seedTestFeedData(t, db)

	postsRepo := repository.NewPostRepository(db)
	usersRepo := repository.NewUserRepository(db)
	followsRepo := repository.NewFollowRepository(db)

	tests := []struct {
		name       string
		viewerID   string
		groupID    string
		mockGroups *mockGroupMembership
		assertErr  func(t *testing.T, err error)
		verify     func(t *testing.T, res *models.PostListResponse)
	}{
		{
			name:     "Success Member Get Group Posts",
			viewerID: "user-1",
			groupID:  "group-1",
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					if groupID == "group-1" && userID == "user-1" {
						return true, nil
					}
					return false, nil
				},
			},
			verify: func(t *testing.T, res *models.PostListResponse) {
				if res.Total != 1 {
					t.Errorf("expected Total 1, got %d", res.Total)
				}
				if len(res.Posts) != 1 {
					t.Fatalf("expected 1 post, got %d", len(res.Posts))
				}
				if res.Posts[0].ID != "post-7" {
					t.Errorf("expected post-7, got %s", res.Posts[0].ID)
				}
			},
		},
		{
			name:     "Failure Non-Member Get Group Posts",
			viewerID: "user-1",
			groupID:  "group-2",
			mockGroups: &mockGroupMembership{
				isMemberFunc: func(groupID, userID string) (bool, error) {
					return false, nil
				},
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrForbidden) {
					t.Errorf("expected ErrForbidden, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var groups GroupMembership = &mockGroupMembership{}
			if tt.mockGroups != nil {
				groups = tt.mockGroups
			}

			svc := NewService(postsRepo, usersRepo, followsRepo, groups)
			res, err := svc.GetGroupPosts(tt.viewerID, tt.groupID, 10, 0)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if res != nil {
					t.Errorf("expected nil response, got %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.verify != nil {
					tt.verify(t, res)
				}
			}
		})
	}
}

func TestCanViewPost(t *testing.T) {
	db := setupTestDB(t)
	seedTestFeedData(t, db)

	postsRepo := repository.NewPostRepository(db)
	usersRepo := repository.NewUserRepository(db)
	followsRepo := repository.NewFollowRepository(db)

	svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{
		isMemberFunc: func(groupID, userID string) (bool, error) {
			if groupID == "group-1" && userID == "user-1" {
				return true, nil
			}
			return false, nil
		},
	})

	// Can user-1 view post-1 (public)? YES
	can, err := svc.CanViewPost("user-1", "post-1")
	if err != nil || !can {
		t.Errorf("expected true, got can=%v, err=%v", can, err)
	}

	// Can user-1 view post-6 (almost private, not following)? NO
	can, err = svc.CanViewPost("user-1", "post-6")
	if err != nil || can {
		t.Errorf("expected false, got can=%v, err=%v", can, err)
	}

	// Can view non-existent post? Should return error
	_, err = svc.CanViewPost("user-1", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent post, got nil")
	}
}

func TestIsPostOwner(t *testing.T) {
	db := setupTestDB(t)
	seedTestFeedData(t, db)

	postsRepo := repository.NewPostRepository(db)
	usersRepo := repository.NewUserRepository(db)
	followsRepo := repository.NewFollowRepository(db)

	svc := NewService(postsRepo, usersRepo, followsRepo, &mockGroupMembership{})

	// Is user-2 owner of post-1? YES
	isOwner, err := svc.IsPostOwner("user-2", "post-1")
	if err != nil || !isOwner {
		t.Errorf("expected true, got isOwner=%v, err=%v", isOwner, err)
	}

	// Is user-1 owner of post-1? NO
	isOwner, err = svc.IsPostOwner("user-1", "post-1")
	if err != nil || isOwner {
		t.Errorf("expected false, got isOwner=%v, err=%v", isOwner, err)
	}

	// Non-existent post
	_, err = svc.IsPostOwner("user-1", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent post, got nil")
	}
}
