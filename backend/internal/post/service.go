package post

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"social-network/internal/repository"
	"social-network/internal/shared/mediavalidate"
	"social-network/internal/shared/paginate"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

const (
	postUploadDir = "uploads/posts"
)

// GroupMembership is the contract post.Service needs from the groups domain. The interface only contains the one fact a post visibility check
// needs from the groups domain.
type GroupMembership interface {
	// IsMember reports whether userID is an accepted member of groupID. Implementations should treat "invited" and "requested" status rows
	// as NOT a member. Only "accepted" counts.
	IsMember(groupID, userID string) (bool, error)
}

type Service interface {
	// CreatePost validates privacy + visible_to/group_id, then saves the post.
	CreatePost(authorID string, req models.CreatePostRequest) (*models.Post, error)
	// GetPost enforces visibility. It returns apperror.NotFound if viewerID can't see it, not Forbidden, so we don't confirm the post's existence to non-viewers.
	GetPost(viewerID, postID string) (*models.PostResponse, error)
	UpdatePost(authorID, postID string, req models.UpdatePostRequest) error
	DeletePost(authorID, postID string) error
	UploadPostImage(authorID, postID string, file multipart.File, header *multipart.FileHeader) (string, error)

	GetFeed(viewerID string, limit, offset int) (*models.PostListResponse, error)
	GetPostsByAuthor(viewerID, authorID string, limit, offset int) (*models.PostListResponse, error)
	// GetGroupPosts returns a group's posts. Caller must already know the
	// viewer is a member.
	GetGroupPosts(viewerID, groupID string, limit, offset int) (*models.PostListResponse, error)

	CanViewPost(viewerID, postID string) (bool, error)
	// IsPostOwner reports whether userID is the author of postID. Used by comment.Service to let a post's owner delete comments on their own post, in addition
	// to commenters deleting their own.
	IsPostOwner(userID, postID string) (bool, error)
}

type service struct {
	posts   repository.PostRepository
	users   repository.UserRepository
	follows repository.FollowRepository
	groups  GroupMembership
}

func NewService(
	posts repository.PostRepository,
	users repository.UserRepository,
	follows repository.FollowRepository,
	groups GroupMembership,
) Service {
	return &service{posts: posts, users: users, follows: follows, groups: groups}
}

func (s *service) CreatePost(authorID string, req models.CreatePostRequest) (*models.Post, error) {
	privacy, err := normalizePrivacy(req.Privacy)
	if err != nil {
		return nil, err
	}
	content := strings.TrimSpace(req.Content)
	// Image-only posts are allowed (matches the DB's content-or-image CHECK). We don't reject empty content here. An image-only post starts with
	// empty content and gets ImageURL attached via UploadPostImage right after creation. The DB CHECK is the actual enforcement point for
	// "must have content or image"; we don't duplicate that check here because at creation time we can't yet know whether an image upload is coming next.

	if err := s.validatePrivacyInvariants(authorID, privacy, req.GroupID, req.VisibleTo); err != nil {
		return nil, err
	}

	p := &models.Post{
		ID:        uuid.Must(uuid.NewV4()).String(),
		UserID:    authorID,
		GroupID:   req.GroupID,
		Content:   content,
		Privacy:   privacy,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.posts.CreatePost(p); err != nil {
		return nil, err
	}

	if privacy == models.PrivacyPrivate && len(req.VisibleTo) > 0 {
		if err := s.posts.SetPrivateViewers(p.ID, req.VisibleTo); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (s *service) GetPost(viewerID, postID string) (*models.PostResponse, error) {
	p, err := s.posts.GetPostByID(postID)
	if err != nil {
		return nil, err
	}

	allowed, err := s.canView(viewerID, p)
	if err != nil {
		return nil, err
	}
	if !allowed {
		// We use NotFound instead of Forbidden so as not to reveal that a private/group post exists to someone who isn't
		// allowed to see it.
		return nil, apperror.NotFound("post not found")
	}

	author, err := s.users.GetUserByID(p.UserID)
	if err != nil {
		return nil, err
	}

	return toPostResponse(p, author), nil
}

func (s *service) UpdatePost(authorID, postID string, req models.UpdatePostRequest) error {
	p, err := s.posts.GetPostByID(postID)
	if err != nil {
		return err
	}
	if p.UserID != authorID {
		// NotFound rather than Forbidden for the same reason as GetPost. No reason to confirm to a non-owner that this post ID exists.
		return apperror.NotFound("post not found")
	}

	privacy, err := normalizePrivacy(req.Privacy)
	if err != nil {
		return err
	}
	content := strings.TrimSpace(req.Content)

	if err := s.validatePrivacyInvariants(authorID, privacy, req.GroupID, req.VisibleTo); err != nil {
		return err
	}

	p.Content = content
	p.Privacy = privacy
	p.GroupID = req.GroupID

	if err := s.posts.UpdatePost(p); err != nil {
		return err
	}

	if privacy == models.PrivacyPrivate {
		return s.posts.SetPrivateViewers(postID, req.VisibleTo)
	}

	return s.posts.SetPrivateViewers(postID, nil)
}

func (s *service) DeletePost(authorID, postID string) error {
	p, err := s.posts.GetPostByID(postID)
	if err != nil {
		return err
	}

	// You can only delete your own posts
	if p.UserID != authorID {
		return apperror.NotFound("post not found")
	}
	return s.posts.DeletePost(postID)
}

func (s *service) UploadPostImage(authorID, postID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	p, err := s.posts.GetPostByID(postID)
	if err != nil {
		return "", err
	}
	if p.UserID != authorID {
		return "", apperror.NotFound("post not found")
	}

	if header.Size > mediavalidate.MaxImageSize {
		return "", apperror.BadInput("image must be under 5 MB")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !mediavalidate.AllowedImageExts[ext] {
		return "", apperror.BadInput("image must be a JPEG, PNG, or GIF")
	}

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return "", apperror.Internal("could not read file")
	}
	if !mediavalidate.IsAllowedImageBytes(buf[:n]) {
		return "", apperror.BadInput("image must be a JPEG, PNG, or GIF")
	}
	if _, err := file.Seek(0, 0); err != nil {
		return "", apperror.Internal("could not process file")
	}

	if err := os.MkdirAll(postUploadDir, 0o755); err != nil {
		return "", apperror.Internal("could not create upload directory")
	}

	filename := fmt.Sprintf("%s_%d%s", postID, time.Now().UnixNano(), ext)
	destPath := filepath.Join(postUploadDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", apperror.Internal("could not save image")
	}
	defer dest.Close()

	if _, err := dest.ReadFrom(file); err != nil {
		_ = os.Remove(destPath)
		return "", apperror.Internal("could not write image")
	}

	p.ImageURL = destPath
	if err := s.posts.UpdatePost(p); err != nil {
		_ = os.Remove(destPath)
		return "", err
	}

	return destPath, nil
}

func (s *service) GetFeed(viewerID string, limit, offset int) (*models.PostListResponse, error) {
	limit, offset = paginate.ClampPagination(limit, offset)

	posts, err := s.posts.GetFeedForUser(viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.posts.GetFeedCountForUser(viewerID)
	if err != nil {
		return nil, err
	}

	responses, err := s.attachAuthors(posts)
	if err != nil {
		return nil, err
	}

	return &models.PostListResponse{Posts: responses, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *service) GetPostsByAuthor(viewerID, authorID string, limit, offset int) (*models.PostListResponse, error) {
	author, err := s.users.GetUserByID(authorID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAuthorPostsVisible(viewerID, author); err != nil {
		return nil, err
	}

	limit, offset = paginate.ClampPagination(limit, offset)

	posts, err := s.posts.GetPostsByAuthor(viewerID, authorID, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.posts.GetPostCountByAuthor(authorID)
	if err != nil {
		return nil, err
	}

	responses, err := s.attachAuthors(posts)
	if err != nil {
		return nil, err
	}

	return &models.PostListResponse{Posts: responses, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *service) ensureAuthorPostsVisible(viewerID string, author *models.User) error {
	if viewerID == author.ID || author.IsPublic {
		return nil
	}

	isFollowing, err := s.follows.IsFollowing(viewerID, author.ID)
	if err != nil {
		return fmt.Errorf("check following: %w", err)
	}
	if isFollowing {
		return nil
	}

	return apperror.Forbidden("this profile is private")
}

func (s *service) GetGroupPosts(viewerID, groupID string, limit, offset int) (*models.PostListResponse, error) {
	member, err := s.isGroupMember(viewerID, groupID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, apperror.Forbidden("you must be a member of this group to view its posts")
	}

	limit, offset = paginate.ClampPagination(limit, offset)

	posts, err := s.posts.GetPostsForGroup(groupID, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.posts.GetPostCountForGroup(groupID)
	if err != nil {
		return nil, err
	}

	responses, err := s.attachAuthors(posts)
	if err != nil {
		return nil, err
	}

	return &models.PostListResponse{Posts: responses, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *service) CanViewPost(viewerID, postID string) (bool, error) {
	p, err := s.posts.GetPostByID(postID)
	if err != nil {
		return false, err
	}

	return s.canView(viewerID, p)
}

func (s *service) IsPostOwner(userID, postID string) (bool, error) {
	p, err := s.posts.GetPostByID(postID)
	if err != nil {
		return false, err
	}
	return p.UserID == userID, nil
}

func (s *service) canView(viewerID string, p *models.Post) (bool, error) {
	if viewerID == p.UserID {
		return true, nil
	}
	switch p.Privacy {
	case models.PrivacyPublic:
		return true, nil
	case models.PrivacyAlmostPrivate:
		return s.follows.IsFollowing(viewerID, p.UserID)
	case models.PrivacyPrivate:
		return s.posts.IsViewerAllowed(p.ID, viewerID)
	case models.PrivacyGroup:
		if p.GroupID == nil {
			// Group ID must exist for privacy "group"
			return false, nil
		}
		return s.isGroupMember(viewerID, *p.GroupID)
	default:
		// Unknown privacy value - Fail closed, not open.
		return false, nil
	}
}

func normalizePrivacy(v string) (string, error) {
	switch v {
	case models.PrivacyPublic, models.PrivacyAlmostPrivate, models.PrivacyPrivate, models.PrivacyGroup:
		return v, nil
	case "":
		return models.PrivacyPublic, nil // default
	default:
		return "", apperror.BadInput("privacy must be public, almost_private, private, or group")
	}
}

// validatePrivacyInvariants enforces the rules the DB CHECK constraints also encode, but does it in the service layer so we get a clear
// apperror.BadInput instead of an opaque SQLite constraint-violation error.
func (s *service) validatePrivacyInvariants(authorID, privacy string, groupID *string, visibleTo []string) error {
	switch privacy {
	case models.PrivacyGroup:
		if groupID == nil || *groupID == "" {
			return apperror.BadInput("group_id is required for group posts")
		}
		member, err := s.isGroupMember(authorID, *groupID)
		if err != nil {
			return fmt.Errorf("check group membership: %w", err)
		}
		if !member {
			return apperror.Forbidden("you must be a member of this group to post in it")
		}
		if len(visibleTo) > 0 {
			return apperror.BadInput("visible_to is not used for group posts")
		}
	default:
		if groupID != nil {
			return apperror.BadInput("group_id can only be set when privacy is 'group'")
		}
		if privacy == models.PrivacyPrivate {
			if err := s.validateViewersAreFollowers(authorID, visibleTo); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *service) isGroupMember(userID, groupID string) (bool, error) {
	if s.groups == nil {
		return false, nil
	}
	return s.groups.IsMember(groupID, userID)
}

// validateViewersAreFollowers enforces "must choose from followers, not non-followers". Rejects the whole request if any listed user isn't a
// confirmed follower of authorID.
func (s *service) validateViewersAreFollowers(authorID string, viewerIDs []string) error {
	for _, viewerID := range viewerIDs {
		isFollower, err := s.follows.IsFollowing(viewerID, authorID)
		if err != nil {
			return fmt.Errorf("validate viewer %s: %w", viewerID, err)
		}
		if !isFollower {
			return apperror.BadInput("can only select your own followers as viewers")
		}
	}
	return nil
}

func (s *service) attachAuthors(posts []*models.Post) ([]*models.PostResponse, error) {
	out := make([]*models.PostResponse, 0, len(posts))
	// Cache lookups within a single page. A feed page can easily contain several posts from the same author.
	cache := make(map[string]*models.User)

	for _, p := range posts {
		author, ok := cache[p.UserID]
		if !ok {
			var err error
			author, err = s.users.GetUserByID(p.UserID)
			if err != nil {
				return nil, err
			}
			cache[p.UserID] = author
		}
		out = append(out, toPostResponse(p, author))
	}

	return out, nil
}

func toPostResponse(p *models.Post, author *models.User) *models.PostResponse {
	return &models.PostResponse{
		ID:        p.ID,
		Author:    author.ToPublic(),
		GroupID:   p.GroupID,
		Content:   p.Content,
		ImageURL:  p.ImageURL,
		Privacy:   p.Privacy,
		CreatedAt: p.CreatedAt,
	}
}
