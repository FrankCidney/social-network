package repository

import (
	"errors"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"testing"
)

func TestCreateComment(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository)
		comment *models.Comment
		wantErr bool
	}{
		{
			name: "success_top_level_comment",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("failed to create post: %v", err)
				}
			},
			comment: &models.Comment{
				ID:        "comment-1",
				PostID:    "post-1",
				UserID:    "user-1",
				Content:   "A comment",
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "success_nested_comment",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("failed to create post: %v", err)
				}
				c := &models.Comment{
					ID:        "comment-1",
					PostID:    "post-1",
					UserID:    "user-1",
					Content:   "A comment",
					CreatedAt: "2026-06-30T12:00:00Z",
				}
				if err := commentRepo.CreateComment(c); err != nil {
					t.Fatalf("failed to create parent comment: %v", err)
				}
			},
			comment: &models.Comment{
				ID:              "comment-2",
				PostID:          "post-1",
				UserID:          "user-1",
				Content:         "A nested comment",
				ParentCommentID: stringPtr("comment-1"),
				CreatedAt:       "2026-06-30T12:05:00Z",
			},
			wantErr: false,
		},
		{
			name: "success_empty_content_reserved_for_image_upload",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("failed to create post: %v", err)
				}
			},
			comment: &models.Comment{
				ID:        "comment-1",
				PostID:    "post-1",
				UserID:    "user-1",
				Content:   "",
				ImageURL:  "",
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "failure_foreign_key_post_id",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
			},
			comment: &models.Comment{
				ID:        "comment-1",
				PostID:    "non-existent-post",
				UserID:    "user-1",
				Content:   "A comment",
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: true, // FOREIGN KEY constraint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)
			commentRepo := NewCommentRepository(db)

			tt.setup(t, userRepo, postRepo, commentRepo)

			err := commentRepo.CreateComment(tt.comment)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateComment() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCommentByID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository)
		id      string
		want    *models.Comment
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("failed to create post: %v", err)
				}
				c := &models.Comment{
					ID:        "comment-1",
					PostID:    "post-1",
					UserID:    "user-1",
					Content:   "A comment",
					CreatedAt: "2026-06-30T12:00:00Z",
				}
				if err := commentRepo.CreateComment(c); err != nil {
					t.Fatalf("failed to create comment: %v", err)
				}
			},
			id: "comment-1",
			want: &models.Comment{
				ID:        "comment-1",
				PostID:    "post-1",
				UserID:    "user-1",
				Content:   "A comment",
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {},
			id:      "non-existent-comment",
			want:    nil,
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)
			commentRepo := NewCommentRepository(db)

			tt.setup(t, userRepo, postRepo, commentRepo)

			got, err := commentRepo.GetCommentByID(tt.id)
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
				if got.ID != tt.want.ID || got.PostID != tt.want.PostID || got.UserID != tt.want.UserID || got.Content != tt.want.Content {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestUpdateComment(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository)
		comment *models.Comment
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("failed to create post: %v", err)
				}
				c := &models.Comment{
					ID:      "comment-1",
					PostID:  "post-1",
					UserID:  "user-1",
					Content: "Original comment",
				}
				if err := commentRepo.CreateComment(c); err != nil {
					t.Fatalf("failed to create comment: %v", err)
				}
			},
			comment: &models.Comment{
				ID:      "comment-1",
				Content: "Updated comment",
			},
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {},
			comment: &models.Comment{ID: "non-existent-comment", Content: "Updated"},
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)
			commentRepo := NewCommentRepository(db)

			tt.setup(t, userRepo, postRepo, commentRepo)

			err := commentRepo.UpdateComment(tt.comment)
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
				got, err := commentRepo.GetCommentByID(tt.comment.ID)
				if err != nil {
					t.Fatalf("failed to get comment: %v", err)
				}
				if got.Content != tt.comment.Content {
					t.Errorf("expected content %q, got %q", tt.comment.Content, got.Content)
				}
			}
		})
	}
}

func TestDeleteComment(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository)
		id      string
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("failed to create post: %v", err)
				}
				c := &models.Comment{
					ID:      "comment-1",
					PostID:  "post-1",
					UserID:  "user-1",
					Content: "Original comment",
				}
				if err := commentRepo.CreateComment(c); err != nil {
					t.Fatalf("failed to create comment: %v", err)
				}
			},
			id:      "comment-1",
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, userRepo UserRepository, postRepo PostRepository, commentRepo CommentRepository) {},
			id:      "non-existent-comment",
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)
			commentRepo := NewCommentRepository(db)

			tt.setup(t, userRepo, postRepo, commentRepo)

			err := commentRepo.DeleteComment(tt.id)
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
				_, err := commentRepo.GetCommentByID(tt.id)
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected comment to be deleted, got: %v", err)
				}
			}
		})
	}
}

func TestGetCommentsForPost(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	postRepo := NewPostRepository(db)
	commentRepo := NewCommentRepository(db)

	createTestUser(t, userRepo, "user-1", "user1@example.com")
	p := &models.Post{ID: "post-1", UserID: "user-1", Content: "Post content", Privacy: models.PrivacyPublic}
	if err := postRepo.CreatePost(p); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	c1 := &models.Comment{ID: "comment-1", PostID: "post-1", UserID: "user-1", Content: "First comment"}
	c2 := &models.Comment{ID: "comment-2", PostID: "post-1", UserID: "user-1", Content: "Second comment"}

	if err := commentRepo.CreateComment(c1); err != nil {
		t.Fatalf("failed to create comment 1: %v", err)
	}
	if err := commentRepo.CreateComment(c2); err != nil {
		t.Fatalf("failed to create comment 2: %v", err)
	}

	// Adjust created_at times so c1 is guaranteed to be older than c2 (ordered by created_at ASC)
	if _, err := db.Exec("UPDATE comments SET created_at = '2026-06-30T12:00:00Z' WHERE id = 'comment-1'"); err != nil {
		t.Fatalf("failed to update created_at for comment-1: %v", err)
	}
	if _, err := db.Exec("UPDATE comments SET created_at = '2026-06-30T12:01:00Z' WHERE id = 'comment-2'"); err != nil {
		t.Fatalf("failed to update created_at for comment-2: %v", err)
	}

	comments, err := commentRepo.GetCommentsForPost("post-1")
	if err != nil {
		t.Fatalf("GetCommentsForPost failed: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != "comment-1" || comments[1].ID != "comment-2" {
		t.Errorf("expected comments ordered comment-1, comment-2; got %s, %s", comments[0].ID, comments[1].ID)
	}
}
