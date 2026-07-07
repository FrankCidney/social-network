package user

import (
	"errors"
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
)

type Service interface {
	GetProfile(viewerID, targetID string) (*models.Profile, error)
	UpdateProfile(userID string, req models.UpdateProfileRequest) error
	SetProfileVisibility(userID string, isPublic bool) error
	UploadAvatar(userID string, file multipart.File, header *multipart.FileHeader) (string, error)
	GetFollowers(viewerID, targetID string, limit, offset int) (*models.FollowListResponse, error)
	GetFollowing(viewerID, targetID string, limit, offset int) (*models.FollowListResponse, error)
	SearchUsers(viewerID, query string, limit int, excludeGroupID string) ([]*models.UserSearchResult, error)
}

type PostCounter interface {
	GetPostCountByAuthor(authorID string) (int, error)
}

const (
	avatarUploadDir    = "uploads/avatars"
	defaultSearchLimit = 8
	maxSearchLimit     = 20
)

type service struct {
	users   repository.UserRepository
	follows repository.FollowRepository
	posts   PostCounter
}

func NewService(users repository.UserRepository, follows repository.FollowRepository, posts ...PostCounter) Service {
	var postCounter PostCounter
	if len(posts) > 0 {
		postCounter = posts[0]
	}

	return &service{users: users, follows: follows, posts: postCounter}
}

func (s *service) GetProfile(viewerID, targetID string) (*models.Profile, error) {
	target, err := s.users.GetUserByID(targetID)
	if err != nil {
		return nil, err
	}
	isOwnProfile, isFollowing, followRequestStatus, canViewFullProfile, err := s.profileVisibility(viewerID, target)
	if err != nil {
		return nil, err
	}

	followerCount := 0
	followingCount := 0
	postCount := 0
	if canViewFullProfile {
		followerCount, err = s.follows.GetFollowerCount(targetID)
		if err != nil {
			return nil, fmt.Errorf("get follower count: %w", err)
		}
		followingCount, err = s.follows.GetFollowingCount(targetID)
		if err != nil {
			return nil, fmt.Errorf("get following count: %w", err)
		}
		postCount, err = s.getPostCount(targetID)
		if err != nil {
			return nil, fmt.Errorf("get post count: %w", err)
		}
	}

	profile := &models.Profile{
		User:                target.ToPublic(),
		FollowerCount:       followerCount,
		FollowingCount:      followingCount,
		PostCount:           postCount,
		CanViewFullProfile:  canViewFullProfile,
		IsOwnProfile:        isOwnProfile,
		IsFollowing:         isFollowing,
		FollowRequestStatus: followRequestStatus,
	}

	if canViewFullProfile {
		if isOwnProfile {
			profile.Email = target.Email
		}
		profile.AboutMe = target.AboutMe
		profile.DOB = target.DOB
	}

	return profile, nil
}

func (s *service) profileVisibility(viewerID string, target *models.User) (bool, bool, string, bool, error) {
	isOwnProfile := viewerID == target.ID
	if isOwnProfile {
		return true, false, "", true, nil
	}

	isFollowing, err := s.follows.IsFollowing(viewerID, target.ID)
	if err != nil {
		return false, false, "", false, fmt.Errorf("check following: %w", err)
	}

	followRequestStatus := ""
	if !isFollowing {
		followRequest, err := s.follows.GetFollowRequest(viewerID, target.ID)
		if err != nil {
			if !errors.Is(err, apperror.ErrNotFound) {
				return false, false, "", false, fmt.Errorf("get follow request: %w", err)
			}
		} else {
			followRequestStatus = followRequest.Status
		}
	}

	canViewFullProfile := target.IsPublic || isFollowing
	return false, isFollowing, followRequestStatus, canViewFullProfile, nil
}

func (s *service) getPostCount(userID string) (int, error) {
	if s.posts == nil {
		return 0, nil
	}

	return s.posts.GetPostCountByAuthor(userID)
}

func (s *service) UpdateProfile(userID string, req models.UpdateProfileRequest) error {
	user, err := s.users.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Only overwrite fields the caller actually sent.
	if v := strings.TrimSpace(req.FirstName); v != "" {
		user.FirstName = v
	}
	if v := strings.TrimSpace(req.LastName); v != "" {
		user.LastName = v
	}
	if req.DOB != "" {
		if _, err := time.Parse("2006-01-02", req.DOB); err != nil {
			return apperror.BadInput("date of birth must be in YYYY-MM-DD format")
		}
		user.DOB = req.DOB
	}
	if v := strings.TrimSpace(req.Nickname); v != "" {
		user.Nickname = v
	}
	// AboutMe can be explicitly set to empty string to clear it. The field is optional
	// and clearing it is a valid update.
	user.AboutMe = req.AboutMe

	if err := s.users.UpdateUser(user); err != nil {
		return err
	}

	if req.IsPublic != nil {
		return s.users.SetProfileVisibility(userID, *req.IsPublic)
	}

	return nil
}

func (s *service) SetProfileVisibility(userID string, isPublic bool) error {
	return s.users.SetProfileVisibility(userID, isPublic)
}

// Avatar upload happens outside of user registration and profile update. It has it's own independent endpoint.

func (s *service) UploadAvatar(userID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > mediavalidate.MaxImageSize {
		return "", apperror.BadInput("avatar must be under 5 MB")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !mediavalidate.AllowedImageExts[ext] {
		return "", apperror.BadInput("avatar must be a JPEG, PNG, or GIF")
	}

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return "", apperror.Internal("could not read file")
	}

	if !mediavalidate.IsAllowedImageBytes(buf[:n]) {
		return "", apperror.BadInput("avatar must be a JPEG, PNG, or GIF")

	}

	if _, err := file.Seek(0, 0); err != nil {
		return "", apperror.Internal("could not process file")
	}

	if err := os.MkdirAll(avatarUploadDir, 0o755); err != nil {
		return "", apperror.Internal("could not create upload directory")
	}

	// Filename: userID + nanosecond timestamp prevents collisions and
	// makes it easy to find old avatars for cleanup later.
	filename := fmt.Sprintf("%s_%d%s", userID, time.Now().UnixNano(), ext)
	destPath := filepath.Join(avatarUploadDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", apperror.Internal("could not save avatar")
	}
	defer dest.Close()

	if _, err := dest.ReadFrom(file); err != nil {
		_ = os.Remove(destPath) // clean up partial write
		return "", apperror.Internal("could not write avatar")
	}

	if err := s.users.UpdateAvatarPath(userID, destPath); err != nil {
		_ = os.Remove(destPath) // clean up orphaned file
		return "", err
	}

	return destPath, nil
}

func (s *service) GetFollowers(viewerID, targetID string, limit, offset int) (*models.FollowListResponse, error) {
	limit, offset = paginate.ClampPagination(limit, offset)

	target, err := s.users.GetUserByID(targetID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureProfileListVisible(viewerID, target); err != nil {
		return nil, err
	}

	users, err := s.follows.GetFollowers(targetID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.follows.GetFollowerCount(targetID)
	if err != nil {
		return nil, err
	}

	return &models.FollowListResponse{
		Users:  toPublicUsers(users),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) GetFollowing(viewerID, targetID string, limit, offset int) (*models.FollowListResponse, error) {
	limit, offset = paginate.ClampPagination(limit, offset)

	target, err := s.users.GetUserByID(targetID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureProfileListVisible(viewerID, target); err != nil {
		return nil, err
	}

	users, err := s.follows.GetFollowing(targetID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.follows.GetFollowingCount(targetID)
	if err != nil {
		return nil, err
	}

	return &models.FollowListResponse{
		Users:  toPublicUsers(users),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *service) ensureProfileListVisible(viewerID string, target *models.User) error {
	isOwnProfile, _, _, canViewFullProfile, err := s.profileVisibility(viewerID, target)
	if err != nil {
		return err
	}
	if isOwnProfile || canViewFullProfile {
		return nil
	}

	return apperror.Forbidden("this profile is private")
}

func (s *service) SearchUsers(viewerID, query string, limit int, excludeGroupID string) ([]*models.UserSearchResult, error) {
	query = strings.TrimSpace(query)

	switch {
	case limit <= 0:
		limit = defaultSearchLimit
	case limit > maxSearchLimit:
		limit = maxSearchLimit
	}

	users, err := s.users.SearchUsers(query, limit, viewerID, strings.TrimSpace(excludeGroupID))
	if err != nil {
		return nil, err
	}

	results := make([]*models.UserSearchResult, 0, len(users))
	for _, user := range users {
		result := &models.UserSearchResult{
			ID:         user.ID,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			Nickname:   user.Nickname,
			AvatarPath: user.AvatarPath,
			IsPublic:   user.IsPublic,
		}

		isFollowing, err := s.follows.IsFollowing(viewerID, user.ID)
		if err != nil {
			return nil, fmt.Errorf("check following: %w", err)
		}
		result.IsFollowing = isFollowing

		if !isFollowing {
			followRequest, err := s.follows.GetFollowRequest(viewerID, user.ID)
			if err != nil {
				if !errors.Is(err, apperror.ErrNotFound) {
					return nil, fmt.Errorf("get follow request: %w", err)
				}
			} else {
				result.FollowRequestStatus = followRequest.Status
			}
		}

		results = append(results, result)
	}

	return results, nil
}

func toPublicUsers(users []*models.User) []*models.PublicUser {
	out := make([]*models.PublicUser, len(users))
	for i, u := range users {
		out[i] = u.ToPublic()
	}
	return out
}
