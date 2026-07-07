package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/response"
	"strconv"
)

// writeServiceError maps domain errors to HTTP status codes
func writeServiceError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		// Unexpected error — don't leak internals
		slog.Error("unhandled error reached handler", "error", err)
		response.Error(w, apperror.Internal("something went wrong"), http.StatusInternalServerError)
		return
	}
 
	var status int
	switch {
	case errors.Is(appErr.Err, apperror.ErrBadInput):
		status = http.StatusBadRequest
	case errors.Is(appErr.Err, apperror.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(appErr.Err, apperror.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(appErr.Err, apperror.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(appErr.Err, apperror.ErrConflict):
		status = http.StatusConflict
	default:
		status = http.StatusInternalServerError
	}
 
	response.Error(w, appErr, status)
}

func parsePagination(r *http.Request) (limit, offset int) {
	// Service layer handles the edge cases for limit and offset
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	return
}