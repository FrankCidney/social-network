package handlers

import (
	"net/http"
	"social-network/internal/middleware"
	"social-network/internal/notification"
	"social-network/internal/response"
)

type NotificationHandler struct {
	service notification.Service
}

func NewNotificationHandler(service notification.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	limit, offset := parsePagination(r)

	result, err := h.service.GetNotifications(userID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	notificationID := r.PathValue("notificationId")

	if err := h.service.MarkAsRead(userID, notificationID); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

func (h *NotificationHandler) MarkAsResolved(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID
	notificationID := r.PathValue("notificationId")

	if err := h.service.MarkAsResolved(userID, notificationID); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserFromContext(r.Context()).ID

	if err := h.service.MarkAllAsRead(userID); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}
