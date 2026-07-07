package follow

import (
	"errors"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"social-network/internal/repository"
)

type Service interface {
	SendFollowRequest(senderID, targetID string) error
	AcceptRequest(recipientID, senderID string) error
	DeclineRequest(recipientID, senderID string) error
	Unfollow(followerID, followedID string) error
	IsFollowing(followerID, followedID string) (bool, error)
	GetPendingRequests(userID string) ([]*models.FollowRequest, error)
}

type NotificationService interface {
	NotifyFollowRequest(senderID, receiverID string) error
	NotifyFollowAccepted(senderID, receiverID string) error
}

type service struct {
	users   repository.UserRepository
	follows repository.FollowRepository
	notify  NotificationService // this can be nil in tests
}

func NewService(users repository.UserRepository, follows repository.FollowRepository, notify NotificationService) Service {
	return &service{users: users, follows: follows, notify: notify}
}

// Checks is_public, then either auto-follows or creates a pending follow request
func (s *service) SendFollowRequest(senderID, targetID string) error {
	if senderID == targetID {
		return apperror.BadInput("cannot follow yourself")
	}
 
	// Check if already following.
	already, err := s.follows.IsFollowing(senderID, targetID)
	if err != nil {
		return err
	}
	if already {
		return apperror.Conflict("already following this user")
	}
 
	// Get target (who the follow request is going to) to check whether their profile is public or privae
	target, err := s.users.GetUserByID(targetID)
	if err != nil {
		return err
	}
 
	if target.IsPublic {
		// Follow automatically, skip writing to follow_requests
		if err := s.follows.CreateFollower(senderID, targetID); err != nil {
			return err
		}
		return nil
	}
 
	// If Private, create a pending follow request
	if err := s.follows.CreateFollowRequest(senderID, targetID); err != nil {
		// Conflict means a pending request already exists
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && errors.Is(appErr.Err, apperror.ErrConflict) {
			return apperror.Conflict("follow request already pending")
		}
		return err
	}
 
	// Signal notification layer. We want to fire and forget. A failed
	// notification shouldn't affect the follow request.
	if s.notify != nil {
		_ = s.notify.NotifyFollowRequest(senderID, targetID)
	}
 
	return nil
}

func (s *service) AcceptRequest(recipientID, senderID string) error {
	if err := s.follows.AcceptFollowRequest(senderID, recipientID); err != nil {
		return err
	}
 
	if s.notify != nil {
		_ = s.notify.NotifyFollowAccepted(senderID, recipientID)
	}
 
	return nil
}

func (s *service) DeclineRequest(recipientID, senderID string) error {
	return s.follows.DeclineFollowRequest(senderID, recipientID)
}
 
func (s *service) Unfollow(followerID, followedID string) error {
	if followerID == followedID {
		return apperror.BadInput("cannot unfollow yourself")
	}
	return s.follows.DeleteFollower(followerID, followedID)
}

func (s *service) IsFollowing(followerID, followedID string) (bool, error) {
	return s.follows.IsFollowing(followerID, followedID)
}

func (s *service) GetPendingRequests(userID string) ([]*models.FollowRequest, error) {
	return s.follows.GetPendingRequests(userID)
}
