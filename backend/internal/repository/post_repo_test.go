package repository

import (
	"database/sql"
	"errors"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"testing"
)

func createGroupHelper(t *testing.T, db interface {
	Exec(string, ...any) (sql.Result, error)
}, id, creatorID, title string) {
	t.Helper()
	const query = `INSERT INTO groups (id, creator_id, title, description, created_at) VALUES (?, ?, ?, 'desc', datetime('now'))`
	_, err := db.Exec(query, id, creatorID, title)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
}

func createGroupMemberHelper(t *testing.T, db interface {
	Exec(string, ...any) (sql.Result, error)
}, groupID, userID, status string) {
	t.Helper()
	const query = `INSERT INTO group_members (group_id, user_id, status, created_at) VALUES (?, ?, ?, datetime('now'))`
	_, err := db.Exec(query, groupID, userID, status)
	if err != nil {
		t.Fatalf("failed to create group member: %v", err)
	}
}

func TestCreatePost(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, db *sql.DB, userRepo UserRepository)
		post    *models.Post
		wantErr bool
	}{
		{
			name: "success_public_post",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
			},
			post: &models.Post{
				ID:        "post-1",
				UserID:    "user-1",
				Content:   "Hello world",
				Privacy:   models.PrivacyPublic,
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "success_group_post",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createGroupHelper(t, db, "group-1", "user-1", "Group 1")
			},
			post: &models.Post{
				ID:        "post-2",
				UserID:    "user-1",
				GroupID:   stringPtr("group-1"),
				Content:   "Hello group",
				Privacy:   models.PrivacyGroup,
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "success_empty_content_reserved_for_image_upload",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
			},
			post: &models.Post{
				ID:        "post-3",
				UserID:    "user-1",
				Content:   "",
				ImageURL:  "",
				Privacy:   models.PrivacyPublic,
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: false,
		},
		{
			name: "failure_group_post_without_group_id",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
			},
			post: &models.Post{
				ID:        "post-4",
				UserID:    "user-1",
				Content:   "Hello",
				Privacy:   models.PrivacyGroup,
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: true, // CHECK constraint: privacy = 'group' AND group_id IS NOT NULL
		},
		{
			name: "failure_public_post_with_group_id",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createGroupHelper(t, db, "group-1", "user-1", "Group 1")
			},
			post: &models.Post{
				ID:        "post-5",
				UserID:    "user-1",
				GroupID:   stringPtr("group-1"),
				Content:   "Hello",
				Privacy:   models.PrivacyPublic,
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: true, // CHECK constraint: privacy != 'group' AND group_id IS NULL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)

			tt.setup(t, db, userRepo)

			err := postRepo.CreatePost(tt.post)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePost() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPostByID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, userRepo UserRepository, postRepo PostRepository)
		id      string
		want    *models.Post
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, userRepo UserRepository, postRepo PostRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{
					ID:        "post-1",
					UserID:    "user-1",
					Content:   "Hello world",
					Privacy:   models.PrivacyPublic,
					CreatedAt: "2026-06-30T12:00:00Z",
				}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("setup failed to create post: %v", err)
				}
			},
			id: "post-1",
			want: &models.Post{
				ID:        "post-1",
				UserID:    "user-1",
				Content:   "Hello world",
				Privacy:   models.PrivacyPublic,
				CreatedAt: "2026-06-30T12:00:00Z",
			},
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, userRepo UserRepository, postRepo PostRepository) {},
			id:      "non-existent-post",
			want:    nil,
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)

			tt.setup(t, userRepo, postRepo)

			got, err := postRepo.GetPostByID(tt.id)
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
				if got.ID != tt.want.ID || got.UserID != tt.want.UserID || got.Content != tt.want.Content || got.Privacy != tt.want.Privacy {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestUpdatePost(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, db *sql.DB, userRepo UserRepository, postRepo PostRepository)
		post    *models.Post
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository, postRepo PostRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				p := &models.Post{
					ID:      "post-1",
					UserID:  "user-1",
					Content: "Original content",
					Privacy: models.PrivacyPublic,
				}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			post: &models.Post{
				ID:      "post-1",
				Content: "Updated content",
				Privacy: models.PrivacyPublic,
			},
			wantErr: nil,
		},
		{
			name:  "not_found",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository, postRepo PostRepository) {},
			post: &models.Post{
				ID:      "non-existent-post",
				Content: "Updated content",
				Privacy: models.PrivacyPublic,
			},
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)

			tt.setup(t, db, userRepo, postRepo)

			err := postRepo.UpdatePost(tt.post)
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
				// Verify update
				got, err := postRepo.GetPostByID(tt.post.ID)
				if err != nil {
					t.Fatalf("failed to retrieve post: %v", err)
				}
				if got.Content != tt.post.Content {
					t.Errorf("expected content %q, got %q", tt.post.Content, got.Content)
				}
			}
		})
	}
}

func TestDeletePost(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, db *sql.DB, userRepo UserRepository, postRepo PostRepository)
		id      string
		wantErr error
	}{
		{
			name: "success_cascades_visibility_and_comments",
			setup: func(t *testing.T, db *sql.DB, userRepo UserRepository, postRepo PostRepository) {
				createTestUser(t, userRepo, "user-1", "user1@example.com")
				createTestUser(t, userRepo, "user-2", "user2@example.com")

				p := &models.Post{
					ID:      "post-1",
					UserID:  "user-1",
					Content: "Private post",
					Privacy: models.PrivacyPrivate,
				}
				if err := postRepo.CreatePost(p); err != nil {
					t.Fatalf("setup failed to create post: %v", err)
				}

				if err := postRepo.SetPrivateViewers("post-1", []string{"user-2"}); err != nil {
					t.Fatalf("setup failed to set viewers: %v", err)
				}

				// Insert a comment
				const insertComment = `INSERT INTO comments (id, post_id, user_id, content, created_at) VALUES ('comment-1', 'post-1', 'user-2', 'Great post!', datetime('now'))`
				if _, err := db.Exec(insertComment); err != nil {
					t.Fatalf("setup failed to create comment: %v", err)
				}
			},
			id:      "post-1",
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, db *sql.DB, userRepo UserRepository, postRepo PostRepository) {},
			id:      "non-existent-post",
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			userRepo := NewUserRepository(db)
			postRepo := NewPostRepository(db)

			tt.setup(t, db, userRepo, postRepo)

			err := postRepo.DeletePost(tt.id)
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
				// Verify post is deleted
				_, err := postRepo.GetPostByID(tt.id)
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected post to be deleted, got: %v", err)
				}
				// Verify visibility is cleared
				var visCount int
				err = db.QueryRow("SELECT COUNT(*) FROM post_visibility WHERE post_id = ?", tt.id).Scan(&visCount)
				if err != nil {
					t.Fatalf("failed to query visibility count: %v", err)
				}
				if visCount != 0 {
					t.Errorf("expected post_visibility to be cleared, got %d", visCount)
				}
				// Verify comments are cleared
				var commentCount int
				err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ?", tt.id).Scan(&commentCount)
				if err != nil {
					t.Fatalf("failed to query comment count: %v", err)
				}
				if commentCount != 0 {
					t.Errorf("expected comments to be cleared, got %d", commentCount)
				}
			}
		})
	}
}

func TestSetPrivateViewersAndIsViewerAllowed(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	postRepo := NewPostRepository(db)

	createTestUser(t, userRepo, "author", "author@example.com")
	createTestUser(t, userRepo, "viewer-1", "v1@example.com")
	createTestUser(t, userRepo, "viewer-2", "v2@example.com")

	p := &models.Post{
		ID:      "post-1",
		UserID:  "author",
		Content: "Private",
		Privacy: models.PrivacyPrivate,
	}
	if err := postRepo.CreatePost(p); err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	// Set viewers
	err := postRepo.SetPrivateViewers("post-1", []string{"viewer-1", "viewer-2"})
	if err != nil {
		t.Fatalf("SetPrivateViewers failed: %v", err)
	}

	// Verify allowed
	allowed1, err := postRepo.IsViewerAllowed("post-1", "viewer-1")
	if err != nil || !allowed1 {
		t.Errorf("expected viewer-1 to be allowed, err: %v, allowed: %v", err, allowed1)
	}

	allowed2, err := postRepo.IsViewerAllowed("post-1", "viewer-2")
	if err != nil || !allowed2 {
		t.Errorf("expected viewer-2 to be allowed, err: %v, allowed: %v", err, allowed2)
	}

	allowed3, err := postRepo.IsViewerAllowed("post-1", "author")
	if err != nil || allowed3 {
		t.Errorf("expected author to NOT be in post_visibility table (only explicit viewers are), allowed: %v", allowed3)
	}

	// Update viewers (replace list)
	err = postRepo.SetPrivateViewers("post-1", []string{"viewer-2"})
	if err != nil {
		t.Fatalf("SetPrivateViewers update failed: %v", err)
	}

	// Verify update
	allowed1, _ = postRepo.IsViewerAllowed("post-1", "viewer-1")
	if allowed1 {
		t.Errorf("expected viewer-1 to be removed")
	}
	allowed2, _ = postRepo.IsViewerAllowed("post-1", "viewer-2")
	if !allowed2 {
		t.Errorf("expected viewer-2 to still be allowed")
	}
}

func TestGetFeedForUserAndCount(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	followRepo := NewFollowRepository(db)
	postRepo := NewPostRepository(db)

	// Users
	createTestUser(t, userRepo, "viewer", "viewer@example.com")
	createTestUser(t, userRepo, "user-public", "pub@example.com")
	createTestUser(t, userRepo, "user-almost", "almost@example.com")
	createTestUser(t, userRepo, "user-private", "priv@example.com")
	createTestUser(t, userRepo, "user-group", "grp@example.com")

	// Set up relationships
	// viewer follows user-almost
	if err := followRepo.CreateFollower("viewer", "user-almost"); err != nil {
		t.Fatalf("failed to create follower: %v", err)
	}

	// Set up group and membership
	createGroupHelper(t, db, "group-1", "user-group", "Group 1")
	createGroupMemberHelper(t, db, "group-1", "viewer", "accepted")

	// Posts
	// 1. Viewer's own post (always visible)
	pOwn := &models.Post{ID: "p-own", UserID: "viewer", Content: "My post", Privacy: models.PrivacyPrivate}
	// 2. Public post (always visible)
	pPub := &models.Post{ID: "p-pub", UserID: "user-public", Content: "Public post", Privacy: models.PrivacyPublic}
	// 3. Almost Private post - viewer is follower (visible)
	pAlmostVis := &models.Post{ID: "p-almost-vis", UserID: "user-almost", Content: "Almost private vis", Privacy: models.PrivacyAlmostPrivate}
	// 4. Almost Private post - viewer is NOT follower (not visible)
	pAlmostNotVis := &models.Post{ID: "p-almost-not", UserID: "user-public", Content: "Almost private not vis", Privacy: models.PrivacyAlmostPrivate}
	// 5. Private post - viewer is allowed (visible)
	pPrivVis := &models.Post{ID: "p-priv-vis", UserID: "user-private", Content: "Private vis", Privacy: models.PrivacyPrivate}
	// 6. Private post - viewer is NOT allowed (not visible)
	pPrivNotVis := &models.Post{ID: "p-priv-not", UserID: "user-private", Content: "Private not vis", Privacy: models.PrivacyPrivate}
	// 7. Group post - viewer is member (visible)
	pGrpVis := &models.Post{ID: "p-grp-vis", UserID: "user-group", GroupID: stringPtr("group-1"), Content: "Group vis", Privacy: models.PrivacyGroup}

	allPosts := []*models.Post{pOwn, pPub, pAlmostVis, pAlmostNotVis, pPrivVis, pPrivNotVis, pGrpVis}
	for _, p := range allPosts {
		if err := postRepo.CreatePost(p); err != nil {
			t.Fatalf("failed to create post %s: %v", p.ID, err)
		}
	}

	// Set private viewers
	if err := postRepo.SetPrivateViewers("p-priv-vis", []string{"viewer"}); err != nil {
		t.Fatalf("failed to set private viewers: %v", err)
	}

	// Adjust created_at times to make sorting deterministic (pOwn > pPub > pAlmostVis > pPrivVis > pGrpVis)
	dates := map[string]string{
		"p-own":        "2026-06-30T12:10:00Z",
		"p-pub":        "2026-06-30T12:08:00Z",
		"p-almost-vis": "2026-06-30T12:06:00Z",
		"p-priv-vis":   "2026-06-30T12:04:00Z",
		"p-grp-vis":    "2026-06-30T12:02:00Z",
	}
	for id, dt := range dates {
		if _, err := db.Exec("UPDATE posts SET created_at = ? WHERE id = ?", dt, id); err != nil {
			t.Fatalf("failed to update created_at for %s: %v", id, err)
		}
	}

	// Get Feed
	feed, err := postRepo.GetFeedForUser("viewer", 10, 0)
	if err != nil {
		t.Fatalf("GetFeedForUser failed: %v", err)
	}

	expectedIDs := []string{"p-own", "p-pub", "p-almost-vis", "p-priv-vis", "p-grp-vis"}
	if len(feed) != len(expectedIDs) {
		t.Errorf("expected %d posts in feed, got %d", len(expectedIDs), len(feed))
	}
	for i, p := range feed {
		if i < len(expectedIDs) && p.ID != expectedIDs[i] {
			t.Errorf("at index %d: expected post %s, got %s", i, expectedIDs[i], p.ID)
		}
	}

	// Count
	count, err := postRepo.GetFeedCountForUser("viewer")
	if err != nil {
		t.Fatalf("GetFeedCountForUser failed: %v", err)
	}
	if count != len(expectedIDs) {
		t.Errorf("expected feed count %d, got %d", len(expectedIDs), count)
	}
}

func TestGetPostsByAuthorAndCount(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	followRepo := NewFollowRepository(db)
	postRepo := NewPostRepository(db)

	createTestUser(t, userRepo, "author", "author@example.com")
	createTestUser(t, userRepo, "viewer-following", "v1@example.com")
	createTestUser(t, userRepo, "viewer-stranger", "v2@example.com")

	// viewer-following follows author
	if err := followRepo.CreateFollower("viewer-following", "author"); err != nil {
		t.Fatalf("failed to create follower: %v", err)
	}

	// Posts by author
	pPub := &models.Post{ID: "p-pub", UserID: "author", Content: "Public", Privacy: models.PrivacyPublic}
	pAlmost := &models.Post{ID: "p-almost", UserID: "author", Content: "Almost Private", Privacy: models.PrivacyAlmostPrivate}
	pPriv := &models.Post{ID: "p-priv", UserID: "author", Content: "Private", Privacy: models.PrivacyPrivate}

	for _, p := range []*models.Post{pPub, pAlmost, pPriv} {
		if err := postRepo.CreatePost(p); err != nil {
			t.Fatalf("failed to create post %s: %v", p.ID, err)
		}
	}

	// Set private viewer
	if err := postRepo.SetPrivateViewers("p-priv", []string{"viewer-following"}); err != nil {
		t.Fatalf("failed to set private viewer: %v", err)
	}

	// Set deterministic created_at
	dates := map[string]string{
		"p-pub":    "2026-06-30T12:05:00Z",
		"p-almost": "2026-06-30T12:04:00Z",
		"p-priv":   "2026-06-30T12:03:00Z",
	}
	for id, dt := range dates {
		if _, err := db.Exec("UPDATE posts SET created_at = ? WHERE id = ?", dt, id); err != nil {
			t.Fatalf("failed to update created_at for %s: %v", id, err)
		}
	}

	// Case 1: Viewer is author themselves (sees all 3)
	feedAuthor, err := postRepo.GetPostsByAuthor("author", "author", 10, 0)
	if err != nil || len(feedAuthor) != 3 {
		t.Errorf("author feed error: %v, len: %d", err, len(feedAuthor))
	}

	// Case 2: Viewer is following (sees public, almost, and private since they are allowed)
	feedFollowing, err := postRepo.GetPostsByAuthor("viewer-following", "author", 10, 0)
	if err != nil || len(feedFollowing) != 3 {
		t.Errorf("following feed error: %v, len: %d", err, len(feedFollowing))
	}

	// Case 3: Viewer is stranger (sees public only)
	feedStranger, err := postRepo.GetPostsByAuthor("viewer-stranger", "author", 10, 0)
	if err != nil {
		t.Fatalf("stranger feed failed: %v", err)
	}
	if len(feedStranger) != 1 || feedStranger[0].ID != "p-pub" {
		t.Errorf("expected stranger to see only public post, got %+v", feedStranger)
	}

	// Count by author (absolute count = 3)
	count, err := postRepo.GetPostCountByAuthor("author")
	if err != nil || count != 3 {
		t.Errorf("expected count 3, got %d, err: %v", count, err)
	}
}

func TestGetPostsForGroupAndCount(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	postRepo := NewPostRepository(db)

	createTestUser(t, userRepo, "user-1", "user1@example.com")
	createGroupHelper(t, db, "group-1", "user-1", "Group 1")

	p1 := &models.Post{ID: "p-1", UserID: "user-1", GroupID: stringPtr("group-1"), Content: "Post 1", Privacy: models.PrivacyGroup}
	p2 := &models.Post{ID: "p-2", UserID: "user-1", GroupID: stringPtr("group-1"), Content: "Post 2", Privacy: models.PrivacyGroup}

	if err := postRepo.CreatePost(p1); err != nil {
		t.Fatalf("failed to create post 1: %v", err)
	}
	if err := postRepo.CreatePost(p2); err != nil {
		t.Fatalf("failed to create post 2: %v", err)
	}

	// Get posts
	posts, err := postRepo.GetPostsForGroup("group-1", 10, 0)
	if err != nil {
		t.Fatalf("GetPostsForGroup failed: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}

	// Count
	count, err := postRepo.GetPostCountForGroup("group-1")
	if err != nil || count != 2 {
		t.Errorf("expected group post count 2, got %d, err: %v", count, err)
	}
}

func stringPtr(s string) *string {
	return &s
}
