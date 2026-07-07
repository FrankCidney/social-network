package handlers

import (
	"encoding/json"
	"net/http"

	"social-network/internal/chat"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/response"
)

type ChatHandler struct {
	service chat.Service
}

func NewChatHandler(service chat.Service) *ChatHandler {
	return &ChatHandler{service: service}
}

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID

	var req chat.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: models.ErrorValue{Message: "invalid request body"}})
		return
	}

	msg, err := h.service.SendMessage(userID, &req)
	if err != nil {
		response.Error(w, err, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusCreated, msg)
}

func (h *ChatHandler) GetPrivateMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	otherUserID := r.PathValue("userId")

	limit, offset := parsePagination(r)

	messages, err := h.service.GetPrivateMessages(userID, otherUserID, limit, offset)
	if err != nil {
		response.Error(w, err, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) GetGroupMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	groupID := r.PathValue("id")

	limit, offset := parsePagination(r)

	messages, err := h.service.GetGroupMessages(userID, groupID, limit, offset)
	if err != nil {
		response.Error(w, err, http.StatusBadRequest)
		return
	}

	response.JSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	conversations, err := h.service.GetConversations(userID)
	if err != nil {
		response.Error(w, err, http.StatusBadRequest)
		return
	}
	response.JSON(w, http.StatusOK, conversations)
}

func (h *ChatHandler) MarkConversationRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	otherUserID := r.PathValue("userId")
	if err := h.service.MarkConversationRead(userID, otherUserID); err != nil {
		response.Error(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
