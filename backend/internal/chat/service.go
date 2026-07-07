package chat

import (
	"fmt"
	"strings"
	"time"

	"social-network/internal/apperror"
	"social-network/internal/models"
	"social-network/internal/repository"
	"social-network/internal/shared/paginate"
	"social-network/internal/websocket"

	"github.com/gofrs/uuid/v5"
)

type SendMessageRequest struct {
	ReceiverID *string `json:"receiver_id,omitempty"`
	GroupID    *string `json:"group_id,omitempty"`
	Content    string  `json:"content"`
}

type Service interface {
	SendMessage(senderID string, req *SendMessageRequest) (*models.Message, error)
	GetPrivateMessages(user1ID, user2ID string, limit, offset int) ([]*models.Message, error)
	GetGroupMessages(userID, groupID string, limit, offset int) ([]*models.Message, error)
	GetConversations(userID string) ([]*models.Conversation, error)
	MarkConversationRead(userID, otherUserID string) error
}

type service struct {
	msgRepo    repository.MessageRepository
	groupRepo  repository.GroupRepository  // for validating group membership
	followRepo repository.FollowRepository // for validating follow status
	notifier   websocket.Notifier
}

func NewService(msgRepo repository.MessageRepository, groupRepo repository.GroupRepository, followRepo repository.FollowRepository, notifier websocket.Notifier) Service {
	return &service{
		msgRepo:    msgRepo,
		groupRepo:  groupRepo,
		followRepo: followRepo,
		notifier:   notifier,
	}
}

func (s *service) SendMessage(senderID string, req *SendMessageRequest) (*models.Message, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperror.BadInput("content cannot be empty")
	}

	if (req.ReceiverID == nil && req.GroupID == nil) || (req.ReceiverID != nil && req.GroupID != nil) {
		return nil, apperror.BadInput("must provide exactly one of receiver_id or group_id")
	}

	m := &models.Message{
		ID:         uuid.Must(uuid.NewV4()).String(),
		SenderID:   senderID,
		ReceiverID: req.ReceiverID,
		GroupID:    req.GroupID,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	// Validate permissions before sending
	if req.ReceiverID != nil {
		// Private chat: must be following or followed by
		isFollowing, err := s.followRepo.IsFollowing(senderID, *req.ReceiverID)
		if err != nil {
			return nil, err
		}
		isFollowed, err := s.followRepo.IsFollowing(*req.ReceiverID, senderID)
		if err != nil {
			return nil, err
		}
		if !isFollowing && !isFollowed {
			return nil, apperror.Forbidden("cannot message users you are not connected to")
		}
	} else if req.GroupID != nil {
		// Group chat: must be an accepted member
		status, err := s.groupRepo.GetGroupMemberStatus(*req.GroupID, senderID)
		if err != nil {
			return nil, err
		}
		if status != "accepted" {
			return nil, apperror.Forbidden("must be a group member to send messages")
		}
	}

	if err := s.msgRepo.CreateMessage(m); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Broadcast via WebSocket
	if s.notifier != nil {
		if m.ReceiverID != nil {
			// Private message: notify receiver and sender.
			s.notifier.NotifyMessage(*m.ReceiverID, m)
			s.notifier.NotifyMessage(m.SenderID, m)
		} else if m.GroupID != nil {
			// Group message: notify all members
			members, err := s.groupRepo.GetGroupMembers(*m.GroupID)
			if err == nil {
				var memberIDs []string
				for _, mem := range members {
					memberIDs = append(memberIDs, mem.ID)
				}
				s.notifier.NotifyGroupMessage(memberIDs, m)
			}
		}
	}

	return m, nil
}

func (s *service) GetPrivateMessages(user1ID, user2ID string, limit, offset int) ([]*models.Message, error) {
	limit, offset = paginate.ClampPagination(limit, offset)
	return s.msgRepo.GetPrivateMessages(user1ID, user2ID, limit, offset)
}

func (s *service) GetGroupMessages(userID, groupID string, limit, offset int) ([]*models.Message, error) {
	limit, offset = paginate.ClampPagination(limit, offset)
	status, err := s.groupRepo.GetGroupMemberStatus(groupID, userID)
	if err != nil {
		return nil, err
	}
	if status != "accepted" {
		return nil, apperror.Forbidden("must be a group member to view messages")
	}

	return s.msgRepo.GetGroupMessages(groupID, limit, offset)
}

func (s *service) GetConversations(userID string) ([]*models.Conversation, error) {
	return s.msgRepo.GetConversations(userID)
}

func (s *service) MarkConversationRead(userID, otherUserID string) error {
	return s.msgRepo.MarkConversationRead(userID, otherUserID)
}
