package notification

import (
	"social-network/internal/models"
	"social-network/internal/repository"
	"social-network/internal/shared/paginate"
	"time"

	"github.com/gofrs/uuid/v5"
)

type Broadcaster interface {
	NotifyUser(userID string, notification *models.Notification)
	NotifyGroup(userIDs []string, notification *models.Notification)
}

type Service interface {
	GetNotifications(userID string, limit, offset int) (*models.NotificationListResponse, error)
	MarkAsRead(userID, notificationID string) error
	MarkAsResolved(userID, notificationID string) error
	MarkAllAsRead(userID string) error
	NotifyUser(userID string, notification *models.Notification) error
	NotifyGroup(userIDs []string, notification *models.Notification) error
	NotifyFollowRequest(senderID, receiverID string) error
	NotifyFollowAccepted(senderID, receiverID string) error
}

type service struct {
	repo        repository.NotificationRepository
	users       repository.UserRepository
	broadcaster Broadcaster
}

func NewService(
	repo repository.NotificationRepository,
	users repository.UserRepository,
	broadcaster Broadcaster,
) Service {
	return &service{repo: repo, users: users, broadcaster: broadcaster}
}

func (s *service) GetNotifications(userID string, limit, offset int) (*models.NotificationListResponse, error) {
	limit, offset = paginate.ClampPagination(limit, offset)

	notifications, err := s.repo.GetUserNotifications(userID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.GetNotificationCount(userID)
	if err != nil {
		return nil, err
	}

	return &models.NotificationListResponse{
		Notifications: notifications,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	}, nil
}

func (s *service) MarkAsRead(userID, notificationID string) error {
	return s.repo.MarkNotificationAsRead(userID, notificationID)
}

func (s *service) MarkAsResolved(userID, notificationID string) error {
	return s.repo.MarkNotificationAsResolved(userID, notificationID)
}

func (s *service) MarkAllAsRead(userID string) error {
	return s.repo.MarkAllNotificationsAsRead(userID)
}

func (s *service) NotifyUser(userID string, notification *models.Notification) error {
	if notification == nil {
		return nil
	}

	next := *notification
	next.ID = uuid.Must(uuid.NewV4()).String()
	next.UserID = userID
	next.IsRead = false
	next.IsResolved = false
	if next.CreatedAt.IsZero() {
		next.CreatedAt = time.Now()
	}
	if next.Actor == nil && s.users != nil && next.ActorID != "" {
		if actor, err := s.users.GetUserByID(next.ActorID); err == nil {
			next.Actor = actor.ToPublic()
		}
	}

	if err := s.repo.CreateNotification(&next); err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.NotifyUser(userID, &next)
	}

	return nil
}

func (s *service) NotifyGroup(userIDs []string, notification *models.Notification) error {
	for _, userID := range userIDs {
		if err := s.NotifyUser(userID, notification); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) NotifyFollowRequest(senderID, receiverID string) error {
	return s.NotifyUser(receiverID, &models.Notification{
		ActorID: senderID,
		Type:    "follow_request",
	})
}

func (s *service) NotifyFollowAccepted(senderID, receiverID string) error {
	return s.NotifyUser(senderID, &models.Notification{
		ActorID: receiverID,
		Type:    "follow_accepted",
	})
}
