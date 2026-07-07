package comment

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
	"testing"
	"time"

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

// fakePostService implements post.Service for testing
type fakePostService struct {
	canViewPost map[string]bool // key: viewerID_postID
	isPostOwner map[string]bool // key: userID_postID
}

func (f *fakePostService) CreatePost(authorID string, req models.CreatePostRequest) (*models.Post, error) {
	return nil, nil
}

func (f *fakePostService) GetPost(viewerID, postID string) (*models.PostResponse, error) {
	return nil, nil
}

func (f *fakePostService) UpdatePost(authorID, postID string, req models.UpdatePostRequest) error {
	return nil
}

func (f *fakePostService) DeletePost(authorID, postID string) error {
	return nil
}

func (f *fakePostService) UploadPostImage(authorID, postID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	return "", nil
}

func (f *fakePostService) GetFeed(viewerID string, limit, offset int) (*models.PostListResponse, error) {
	return nil, nil
}

func (f *fakePostService) GetPostsByAuthor(viewerID, authorID string, limit, offset int) (*models.PostListResponse, error) {
	return nil, nil
}

func (f *fakePostService) GetGroupPosts(viewerID, groupID string, limit, offset int) (*models.PostListResponse, error) {
	return nil, nil
}

func (f *fakePostService) CanViewPost(viewerID, postID string) (bool, error) {
	key := viewerID + "_" + postID
	if can, ok := f.canViewPost[key]; ok {
		return can, nil
	}
	return false, nil
}

func (f *fakePostService) IsPostOwner(userID, postID string) (bool, error) {
	key := userID + "_" + postID
	if isOwner, ok := f.isPostOwner[key]; ok {
		return isOwner, nil
	}
	return false, nil
}

type fakeMultipartFile struct {
	*bytes.Reader
}

func (f fakeMultipartFile) Close() error { return nil }

// helper to create a valid user
func createUser(t *testing.T, users repository.UserRepository, id, email string) {
	t.Helper()
	err := users.CreateUser(&models.User{
		ID:        id,
		Email:     email,
		Password:  "password",
		FirstName: "First",
		LastName:  "Last",
		DOB:       "1990-01-01",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to create user %s: %v", id, err)
	}
}

// helper to create a valid post
func createPost(t *testing.T, posts repository.PostRepository, id, userID string) {
	t.Helper()
	err := posts.CreatePost(&models.Post{
		ID:        id,
		UserID:    userID,
		Content:   "non-empty post content",
		Privacy:   models.PrivacyPublic,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to create post %s: %v", id, err)
	}
}

// helper to create a valid comment
func createComment(t *testing.T, comments repository.CommentRepository, id, postID, userID string) {
	t.Helper()
	err := comments.CreateComment(&models.Comment{
		ID:        id,
		PostID:    postID,
		UserID:    userID,
		Content:   "non-empty comment content",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to create comment %s: %v", id, err)
	}
}

func TestAddComment(t *testing.T) {
	const (
		authorID = "author-123"
		postID   = "post-456"
	)

	tests := []struct {
		name      string
		authorID  string
		postID    string
		req       models.CreateCommentRequest
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository)
		setupPost func(t *testing.T, fake *fakePostService)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:     "Success top-level comment",
			authorID: authorID,
			postID:   postID,
			req:      models.CreateCommentRequest{Content: "Great post!"},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[authorID+"_"+postID] = true
			},
		},
		{
			name:     "Success reply to comment",
			authorID: authorID,
			postID:   postID,
			req:      models.CreateCommentRequest{Content: "Reply", ParentCommentID: ptr("parent-789")},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
				createComment(t, comments, "parent-789", postID, authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[authorID+"_"+postID] = true
			},
		},
		{
			name:     "Post not found (cannot view)",
			authorID: authorID,
			postID:   postID,
			req:      models.CreateCommentRequest{Content: "Test"},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[authorID+"_"+postID] = false
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Parent comment does not belong to post",
			authorID: authorID,
			postID:   postID,
			req:      models.CreateCommentRequest{Content: "Reply", ParentCommentID: ptr("parent-other")},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
				createPost(t, posts, "other-post", authorID)
				createComment(t, comments, "parent-other", "other-post", authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[authorID+"_"+postID] = true
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Parent comment not found",
			authorID: authorID,
			postID:   postID,
			req:      models.CreateCommentRequest{Content: "Reply", ParentCommentID: ptr("nonexistent")},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[authorID+"_"+postID] = true
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Database error on CreateComment",
			authorID: authorID,
			postID:   postID,
			req:      models.CreateCommentRequest{Content: "Test"},
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
				_, _ = db.Exec("DROP TABLE comments")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[authorID+"_"+postID] = true
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
			posts := repository.NewPostRepository(db)
			comments := repository.NewCommentRepository(db)

			fakePost := &fakePostService{
				canViewPost: make(map[string]bool),
			}
			if tt.setupPost != nil {
				tt.setupPost(t, fakePost)
			}

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, posts, comments)
			}

			svc := NewService(comments, users, fakePost)
			_, err := svc.AddComment(tt.authorID, tt.postID, tt.req)

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

func TestGetCommentTree(t *testing.T) {
	const (
		viewerID = "viewer-123"
		postID   = "post-456"
	)

	tests := []struct {
		name      string
		viewerID  string
		postID    string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository)
		setupPost func(t *testing.T, fake *fakePostService)
		assertErr func(t *testing.T, err error)
		assertRes func(t *testing.T, tree []*models.CommentResponse)
	}{
		{
			name:     "Success empty tree",
			viewerID: viewerID,
			postID:   postID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, viewerID, "viewer@example.com")
				createPost(t, posts, postID, viewerID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[viewerID+"_"+postID] = true
			},
			assertRes: func(t *testing.T, tree []*models.CommentResponse) {
				if len(tree) != 0 {
					t.Errorf("expected empty tree, got %d comments", len(tree))
				}
			},
		},
		{
			name:     "Success with top-level comments",
			viewerID: viewerID,
			postID:   postID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, viewerID, "viewer@example.com")
				createUser(t, users, "user-1", "user1@example.com")
				createPost(t, posts, postID, viewerID)
				createComment(t, comments, "comment-1", postID, "user-1")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[viewerID+"_"+postID] = true
			},
			assertRes: func(t *testing.T, tree []*models.CommentResponse) {
				if len(tree) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(tree))
				}
			},
		},
		{
			name:     "Success with nested replies",
			viewerID: viewerID,
			postID:   postID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, viewerID, "viewer@example.com")
				createUser(t, users, "user-1", "user1@example.com")
				createUser(t, users, "user-2", "user2@example.com")
				createPost(t, posts, postID, viewerID)
				parentID := "parent-1"
				createComment(t, comments, parentID, postID, "user-1")
				createComment(t, comments, "reply-1", postID, "user-2")
				// Update reply to have parent
				_, _ = db.Exec("UPDATE comments SET parent_comment_id = ? WHERE id = ?", parentID, "reply-1")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[viewerID+"_"+postID] = true
			},
			assertRes: func(t *testing.T, tree []*models.CommentResponse) {
				if len(tree) != 1 {
					t.Fatalf("expected 1 top-level comment, got %d", len(tree))
				}
				if len(tree[0].Replies) != 1 {
					t.Errorf("expected 1 reply, got %d", len(tree[0].Replies))
				}
			},
		},
		{
			name:     "Post not found (cannot view)",
			viewerID: viewerID,
			postID:   postID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, viewerID, "viewer@example.com")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[viewerID+"_"+postID] = false
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:     "Database error on GetCommentsForPost",
			viewerID: viewerID,
			postID:   postID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, viewerID, "viewer@example.com")
				_, _ = db.Exec("DROP TABLE comments")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.canViewPost[viewerID+"_"+postID] = true
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
			posts := repository.NewPostRepository(db)
			comments := repository.NewCommentRepository(db)

			fakePost := &fakePostService{
				canViewPost: make(map[string]bool),
			}
			if tt.setupPost != nil {
				tt.setupPost(t, fakePost)
			}

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, posts, comments)
			}

			svc := NewService(comments, users, fakePost)
			tree, err := svc.GetCommentTree(tt.viewerID, tt.postID)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.assertRes != nil {
					tt.assertRes(t, tree)
				}
			}
		})
	}
}

func TestDeleteComment(t *testing.T) {
	const (
		authorID  = "author-123"
		otherID   = "other-456"
		postID    = "post-789"
		commentID = "comment-123"
	)

	tests := []struct {
		name      string
		authorID  string
		commentID string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository)
		setupPost func(t *testing.T, fake *fakePostService)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:      "Author deletes own comment",
			authorID:  authorID,
			commentID: commentID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
				createComment(t, comments, commentID, postID, authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.isPostOwner[authorID+"_"+postID] = false
			},
		},
		{
			name:      "Post owner deletes comment",
			authorID:  otherID,
			commentID: commentID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, otherID, "other@example.com")
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, otherID)
				createComment(t, comments, commentID, postID, authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.isPostOwner[otherID+"_"+postID] = true
			},
		},
		{
			name:      "Non-author non-owner forbidden",
			authorID:  otherID,
			commentID: commentID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, otherID, "other@example.com")
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, postID, authorID)
				createComment(t, comments, commentID, postID, authorID)
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
				fake.isPostOwner[otherID+"_"+postID] = false
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrForbidden) {
					t.Errorf("expected ErrForbidden, got %v", err)
				}
			},
		},
		{
			name:      "Comment not found",
			authorID:  authorID,
			commentID: "nonexistent",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:      "Database error on GetCommentByID",
			authorID:  authorID,
			commentID: commentID,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				_, _ = db.Exec("DROP TABLE comments")
			},
			setupPost: func(t *testing.T, fake *fakePostService) {
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
			posts := repository.NewPostRepository(db)
			comments := repository.NewCommentRepository(db)

			fakePost := &fakePostService{
				isPostOwner: make(map[string]bool),
			}
			if tt.setupPost != nil {
				tt.setupPost(t, fakePost)
			}

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, posts, comments)
			}

			svc := NewService(comments, users, fakePost)
			err := svc.DeleteComment(tt.authorID, tt.commentID)

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

func TestUploadCommentImage(t *testing.T) {
	defer os.RemoveAll("uploads")

	const (
		authorID  = "author-123"
		commentID = "comment-456"
	)

	validPngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	tests := []struct {
		name      string
		authorID  string
		commentID string
		fileBytes []byte
		filename  string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository)
		assertErr func(t *testing.T, err error)
		assertRes func(t *testing.T, path string, comments repository.CommentRepository)
	}{
		{
			name:      "Success upload png",
			authorID:  authorID,
			commentID: commentID,
			fileBytes: validPngBytes,
			filename:  "image.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, "post-123", authorID)
				createComment(t, comments, commentID, "post-123", authorID)
			},
			assertRes: func(t *testing.T, path string, comments repository.CommentRepository) {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("expected file %q to be created, but it does not exist", path)
				}
				c, err := comments.GetCommentByID(commentID)
				if err != nil {
					t.Fatalf("failed to get comment: %v", err)
				}
				if c.ImageURL != path {
					t.Errorf("expected DB image URL to be %q, got %q", path, c.ImageURL)
				}
			},
		},
		{
			name:      "Too large file",
			authorID:  authorID,
			commentID: commentID,
			fileBytes: make([]byte, 6<<20), // 6 MB
			filename:  "image.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, "post-123", authorID)
				createComment(t, comments, commentID, "post-123", authorID)
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Invalid extension",
			authorID:  authorID,
			commentID: commentID,
			fileBytes: validPngBytes,
			filename:  "image.txt",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, "post-123", authorID)
				createComment(t, comments, commentID, "post-123", authorID)
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Invalid image bytes",
			authorID:  authorID,
			commentID: commentID,
			fileBytes: []byte("not-a-png-or-jpeg-or-gif"),
			filename:  "image.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, "post-123", authorID)
				createComment(t, comments, commentID, "post-123", authorID)
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:      "Comment not found",
			authorID:  authorID,
			commentID: "nonexistent",
			fileBytes: validPngBytes,
			filename:  "image.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:      "Not comment author",
			authorID:  "other-user",
			commentID: commentID,
			fileBytes: validPngBytes,
			filename:  "image.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createUser(t, users, "other-user", "other@example.com")
				createPost(t, posts, "post-123", authorID)
				createComment(t, comments, commentID, "post-123", authorID)
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:      "Database error on UpdateComment",
			authorID:  authorID,
			commentID: commentID,
			fileBytes: validPngBytes,
			filename:  "image.png",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, posts repository.PostRepository, comments repository.CommentRepository) {
				createUser(t, users, authorID, "author@example.com")
				createPost(t, posts, "post-123", authorID)
				createComment(t, comments, commentID, "post-123", authorID)
				_, _ = db.Exec("DROP TABLE comments")
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
			posts := repository.NewPostRepository(db)
			comments := repository.NewCommentRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, posts, comments)
			}

			file := fakeMultipartFile{Reader: bytes.NewReader(tt.fileBytes)}
			header := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     int64(len(tt.fileBytes)),
			}

			svc := NewService(comments, users, nil)
			path, err := svc.UploadCommentImage(tt.authorID, tt.commentID, file, header)

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
					tt.assertRes(t, path, comments)
				}
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}
