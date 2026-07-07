package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

type CommentRepository interface {
	CreateComment(c *models.Comment) error
	GetCommentByID(id string) (*models.Comment, error)
	UpdateComment(c *models.Comment) error
	DeleteComment(id string) error
	GetCommentsForPost(postID string) ([]*models.Comment, error)
}

type sqliteCommentRepo struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) CommentRepository {
	return &sqliteCommentRepo{db: db}
}

func (r *sqliteCommentRepo) CreateComment(c *models.Comment) error {
	const query = `
		INSERT INTO comments (id, post_id, user_id, content, image_url, parent_comment_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
 
	_, err := r.db.Exec(query,
		c.ID, c.PostID, c.UserID, c.Content, nullableString(c.ImageURL), c.ParentCommentID, c.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *sqliteCommentRepo) GetCommentByID(id string) (*models.Comment, error) {
	const query = `
		SELECT id, post_id, user_id, content, COALESCE(image_url, ''), parent_comment_id, created_at
		FROM comments WHERE id = ?`
 
	c := &models.Comment{}
	err := r.db.QueryRow(query, id).Scan(
		&c.ID, &c.PostID, &c.UserID, &c.Content, &c.ImageURL, &c.ParentCommentID, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("comment not found")
		}
		return nil, fmt.Errorf("get comment by id: %w", err)
	}
	return c, nil
}

func (r *sqliteCommentRepo) UpdateComment(c *models.Comment) error {
	const query = `UPDATE comments SET content = ?, image_url = ? WHERE id = ?`
	res, err := r.db.Exec(query, nullableString(c.Content), nullableString(c.ImageURL), c.ID)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	return requireOneRow(res, "comment")
}

func (r *sqliteCommentRepo) DeleteComment(id string) error {
	const query = `DELETE FROM comments WHERE id = ?`
	res, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return requireOneRow(res, "comment")
}

func (r *sqliteCommentRepo) GetCommentsForPost(postID string) ([]*models.Comment, error) {
	const query = `
		SELECT id, post_id, user_id, content, COALESCE(image_url, ''), parent_comment_id, created_at
		FROM comments
		WHERE post_id = ?
		ORDER BY created_at ASC`
 
	rows, err := r.db.Query(query, postID)
	if err != nil {
		return nil, fmt.Errorf("get comments for post: %w", err)
	}
	defer rows.Close()
 
	var comments []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID, &c.Content, &c.ImageURL, &c.ParentCommentID, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return comments, nil
}
