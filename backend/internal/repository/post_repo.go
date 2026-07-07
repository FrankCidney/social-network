package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

type PostRepository interface {
	CreatePost(p *models.Post) error
	GetPostByID(id string) (*models.Post, error)
	UpdatePost(p *models.Post) error
	DeletePost(id string) error

	SetPrivateViewers(postID string, userIDs []string) error
	IsViewerAllowed(postID, viewerID string) (bool, error)

	GetFeedForUser(viewerID string, limit, offset int) ([]*models.Post, error)
	GetFeedCountForUser(viewerID string) (int, error)

	GetPostsByAuthor(viewerID, authorID string, limit, offset int) ([]*models.Post, error)
	GetPostCountByAuthor(authorID string) (int, error)

	GetPostsForGroup(groupID string, limit, offset int) ([]*models.Post, error)
	GetPostCountForGroup(groupID string) (int, error)
}

type sqlitePostRepo struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) PostRepository {
	return &sqlitePostRepo{db: db}
}

func (r *sqlitePostRepo) CreatePost(p *models.Post) error {
	const query = `
		INSERT INTO posts (id, user_id, group_id, content, image_url, privacy, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		p.ID, p.UserID, p.GroupID, p.Content, nullableString(p.ImageURL), p.Privacy, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	return nil
}

func (r *sqlitePostRepo) GetPostByID(id string) (*models.Post, error) {
	const query = `
		SELECT id, user_id, group_id, COALESCE(content, ''), COALESCE(image_url, ''), privacy, created_at
		FROM posts WHERE id = ?`

	p := &models.Post{}
	err := r.db.QueryRow(query, id).Scan(
		&p.ID, &p.UserID, &p.GroupID, &p.Content, &p.ImageURL, &p.Privacy, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("post not found")
		}
		return nil, fmt.Errorf("get post by id: %w", err)
	}
	return p, nil
}

func (r *sqlitePostRepo) UpdatePost(p *models.Post) error {
	const query = `
		UPDATE posts SET content = ?, image_url = ?, privacy = ?, group_id = ?
		WHERE id = ?`

	res, err := r.db.Exec(query, nullableString(p.Content), nullableString(p.ImageURL), p.Privacy, p.GroupID, p.ID)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}

	return requireOneRow(res, "post")
}

func (r *sqlitePostRepo) DeletePost(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// We clean up related data ourselves because SQLite won’t always
	// do it automatically unless foreign key support is explicitly enabled.
	if _, err := tx.Exec(`DELETE FROM post_visibility WHERE post_id = ?`, id); err != nil {
		return fmt.Errorf("delete post visibility: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM comments WHERE post_id = ?`, id); err != nil {
		return fmt.Errorf("delete post comments: %w", err)
	}

	res, err := tx.Exec(`DELETE FROM posts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if n == 0 {
		return apperror.NotFound("post not found")
	}

	return tx.Commit()
}

func (r *sqlitePostRepo) SetPrivateViewers(postID string, userIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM post_visibility WHERE post_id = ?`, postID); err != nil {
		return fmt.Errorf("clear post visibility: %w", err)
	}

	const insert = `INSERT INTO post_visibility (post_id, user_id) VALUES (?, ?)`
	for _, uid := range userIDs {
		if _, err := tx.Exec(insert, postID, uid); err != nil {
			return fmt.Errorf("insert post visibility: %w", err)
		}
	}

	return tx.Commit()
}

func (r *sqlitePostRepo) IsViewerAllowed(postID, viewerID string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM post_visibility WHERE post_id = ? AND user_id = ?
		)`

	var exists bool
	if err := r.db.QueryRow(query, postID, viewerID).Scan(&exists); err != nil {
		return false, fmt.Errorf("is viewer allowed: %w", err)
	}

	return exists, nil
}

func (r *sqlitePostRepo) GetFeedForUser(viewerID string, limit, offset int) ([]*models.Post, error) {
	const query = `
		SELECT p.id, p.user_id, p.group_id, COALESCE(p.content, ''), COALESCE(p.image_url, ''), p.privacy, p.created_at
		FROM posts p
		WHERE
			p.user_id = ?
			OR p.privacy = 'public'
			OR (
				p.privacy = 'almost_private'
				AND EXISTS (
					SELECT 1 FROM followers f
					WHERE f.follower_id = ? AND f.followed_id = p.user_id
				)
			)
			OR (
				p.privacy = 'private'
				AND EXISTS (
					SELECT 1 FROM post_visibility pv
					WHERE pv.post_id = p.id AND pv.user_id = ?
				)
			)
			OR (
				p.privacy = 'group'
				AND EXISTS (
					SELECT 1 FROM group_members gm
					WHERE gm.group_id = p.group_id AND gm.user_id = ? AND gm.status = 'accepted'
				)
			)
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?`

	return r.scanPosts(query, viewerID, viewerID, viewerID, viewerID, limit, offset)
}

func (r *sqlitePostRepo) GetFeedCountForUser(viewerID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM posts p
		WHERE
			p.user_id = ?
			OR p.privacy = 'public'
			OR (
				p.privacy = 'almost_private'
				AND EXISTS (
					SELECT 1 FROM followers f
					WHERE f.follower_id = ? AND f.followed_id = p.user_id
				)
			)
			OR (
				p.privacy = 'private'
				AND EXISTS (
					SELECT 1 FROM post_visibility pv
					WHERE pv.post_id = p.id AND pv.user_id = ?
				)
			)
			OR (
				p.privacy = 'group'
				AND EXISTS (
					SELECT 1 FROM group_members gm
					WHERE gm.group_id = p.group_id AND gm.user_id = ? AND gm.status = 'accepted'
				)
			)`

	var count int
	if err := r.db.QueryRow(query, viewerID, viewerID, viewerID, viewerID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get feed count: %w", err)
	}
	return count, nil
}

func (r *sqlitePostRepo) GetPostsByAuthor(viewerID, authorID string, limit, offset int) ([]*models.Post, error) {
	const query = `
		SELECT p.id, p.user_id, p.group_id, COALESCE(p.content, ''), COALESCE(p.image_url, ''), p.privacy, p.created_at
		FROM posts p
		WHERE
			p.user_id = ?
			AND (
				p.user_id = ?
				OR p.privacy = 'public'
				OR (
					p.privacy = 'almost_private'
					AND EXISTS (
						SELECT 1 FROM followers f
						WHERE f.follower_id = ? AND f.followed_id = p.user_id
					)
				)
				OR (
					p.privacy = 'private'
					AND EXISTS (
						SELECT 1 FROM post_visibility pv
						WHERE pv.post_id = p.id AND pv.user_id = ?
					)
				)
				OR (
					p.privacy = 'group'
					AND EXISTS (
						SELECT 1 FROM group_members gm
						WHERE gm.group_id = p.group_id AND gm.user_id = ? AND gm.status = 'accepted'
					)
				)
			)
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?`

	return r.scanPosts(query, authorID, viewerID, viewerID, viewerID, viewerID, limit, offset)
}

func (r *sqlitePostRepo) GetPostCountByAuthor(authorID string) (int, error) {
	const query = `SELECT COUNT(*) FROM posts WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, authorID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get post count by author: %w", err)
	}
	return count, nil
}

func (r *sqlitePostRepo) GetPostsForGroup(groupID string, limit, offset int) ([]*models.Post, error) {
	const query = `
		SELECT id, user_id, group_id, COALESCE(content, ''), COALESCE(image_url, ''), privacy, created_at
		FROM posts
		WHERE group_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	return r.scanPosts(query, groupID, limit, offset)
}

func (r *sqlitePostRepo) GetPostCountForGroup(groupID string) (int, error) {
	const query = `SELECT COUNT(*) FROM posts WHERE group_id = ?`
	var count int
	if err := r.db.QueryRow(query, groupID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get post count for group: %w", err)
	}
	return count, nil
}

func (r *sqlitePostRepo) scanPosts(query string, args ...any) ([]*models.Post, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		p := &models.Post{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.GroupID, &p.Content, &p.ImageURL, &p.Privacy, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return posts, nil
}

