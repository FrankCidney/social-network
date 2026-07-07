package handlers

import (
	"net/http"
	"social-network/internal/follow"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/response"
)

type FollowHandler struct {
	followService follow.Service
}

func NewFollowHandler(followService follow.Service) *FollowHandler {
	return &FollowHandler{followService: followService}
}

// POST /api/follow/{id}
func (h *FollowHandler) SendFollowRequest(w http.ResponseWriter, r *http.Request) {
	sender := middleware.UserFromContext(r.Context())
	targetID := r.PathValue("id")
 
	if err := h.followService.SendFollowRequest(sender.ID, targetID); err != nil {
		writeServiceError(w, err)
		return
	}
	
	response.NoContent(w)
}

// DELETE /api/follow/{id}
func (h *FollowHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	follower := middleware.UserFromContext(r.Context())
	followedID := r.PathValue("id")
 
	if err := h.followService.Unfollow(follower.ID, followedID); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// POST /api/follow/{id}/accept
func (h *FollowHandler) AcceptRequest(w http.ResponseWriter, r *http.Request) {
	recipient := middleware.UserFromContext(r.Context())
	senderID := r.PathValue("id")
 
	if err := h.followService.AcceptRequest(recipient.ID, senderID); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// POST /api/follow/{id}/decline
func (h *FollowHandler) DeclineRequest(w http.ResponseWriter, r *http.Request) {
	recipient := middleware.UserFromContext(r.Context())
	senderID := r.PathValue("id")
 
	if err := h.followService.DeclineRequest(recipient.ID, senderID); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// GET /api/follow/requests
func (h *FollowHandler) GetPendingRequests(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
 
	requests, err := h.followService.GetPendingRequests(u.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toFollowRequestResponses(requests))
}

func toFollowRequestResponses(requests []*models.FollowRequest) []models.FollowRequestResponse {
	out := make([]models.FollowRequestResponse, len(requests))
	for i, fr := range requests {
		out[i] = models.FollowRequestResponse{
			SenderID: fr.SenderID,
			ReceiverID: fr.ReceiverID,
			Status: fr.Status,
		}
	}

	return out
}
 