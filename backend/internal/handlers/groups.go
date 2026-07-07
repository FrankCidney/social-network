package handlers

import (
	"encoding/json"
	"net/http"
	"social-network/internal/groups"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/response"
)

type GroupHandler struct {
	service groups.Service
}

func NewGroupHandler(service groups.Service) *GroupHandler {
	return &GroupHandler{service: service}
}

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: models.ErrorValue{Message: "invalid request body"}})
		return
	}

	group, err := h.service.CreateGroup(userID, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, group)
}

func (h *GroupHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.GetGroups()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	id := r.PathValue("id")
	group, err := h.service.GetGroup(userID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, group)
}

func (h *GroupHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	members, err := h.service.GetMembers(userID, groupID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, members)
}

func (h *GroupHandler) RequestJoin(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	if err := h.service.RequestJoin(groupID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "join request sent"})
}

func (h *GroupHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	var req models.InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: models.ErrorValue{Message: "invalid request body"}})
		return
	}

	if err := h.service.InviteUser(groupID, userID, req.InviteeID); err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "invite sent"})
}

func (h *GroupHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	if err := h.service.AcceptInvite(groupID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *GroupHandler) DeclineInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	if err := h.service.DeclineInvite(groupID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *GroupHandler) GetJoinRequests(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	requests, err := h.service.GetJoinRequests(userID, groupID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, requests)
}

func (h *GroupHandler) AcceptJoinRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")
	targetUserID := r.PathValue("userId")

	if err := h.service.AcceptJoinRequest(groupID, userID, targetUserID); err != nil {
		writeServiceError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *GroupHandler) DeclineJoinRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")
	targetUserID := r.PathValue("userId")

	if err := h.service.DeclineJoinRequest(groupID, userID, targetUserID); err != nil {
		writeServiceError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *GroupHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: models.ErrorValue{Message: "invalid request body"}})
		return
	}

	event, err := h.service.CreateEvent(userID, groupID, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, event)
}

func (h *GroupHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")
	events, err := h.service.GetGroupEvents(groupID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, events)
}

func (h *GroupHandler) RSVPEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	eventID := r.PathValue("id")

	var req models.EventRSVPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: models.ErrorValue{Message: "invalid request body"}})
		return
	}

	if err := h.service.RSVPEvent(eventID, userID, req.Status); err != nil {
		writeServiceError(w, err)
		return
	}
	response.NoContent(w)
}
