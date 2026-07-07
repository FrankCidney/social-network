package db

import (
	"context"
	"social-network/internal/models"
)

// CreatePost inserts a new post into the database
func (s *SQLiteStore) CreatePost(ctx context.Context, post *models.Post) error {
	query := `
		INSERT INTO posts (id, user_id, group_id, content, image_url, privacy, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query,
		post.ID,
		post.UserID,
		post.GroupID,
		post.Content,
		post.ImageURL,
		post.Privacy,
	)
	return err
}

// GetPostByID retrieves a specific post
func (s *SQLiteStore) GetPostByID(ctx context.Context, postID string) (*models.Post, error) {
	query := `
		SELECT id, user_id, group_id, content, image_url, privacy, created_at
		FROM posts WHERE id = ?
	`
	post := &models.Post{}
	err := s.QueryRowContext(ctx, query, postID).Scan(
		&post.ID,
		&post.UserID,
		&post.GroupID,
		&post.Content,
		&post.ImageURL,
		&post.Privacy,
		&post.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return post, nil
}

// CreateComment inserts a new comment for a post
func (s *SQLiteStore) CreateComment(ctx context.Context, comment *models.Comment) error {
	query := `
		INSERT INTO comments (id, post_id, user_id, content, image_url, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query,
		comment.ID,
		comment.PostID,
		comment.UserID,
		comment.Content,
		comment.ImageURL,
	)
	return err
}

// GetCommentsByPostID retrieves all comments for a specific post
func (s *SQLiteStore) GetCommentsByPostID(ctx context.Context, postID string) ([]*models.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, image_url, created_at
		FROM comments WHERE post_id = ? ORDER BY created_at ASC
	`
	rows, err := s.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.ImageURL, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}
