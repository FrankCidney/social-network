package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

// JSON writes v as a JSON body, with the given status code
func JSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("response.JSON: encode failed", "error", err)
        return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

// Error writes a models.ErrorResponse JSON body.
func Error(w http.ResponseWriter, err error, status int) {
	var appErr *apperror.AppError

	if errors.As(err, &appErr) {
		JSON(w, status, models.ErrorResponse{
			Error: models.ErrorValue{
				Code: appErr.Code,
				Message: appErr.Message,
			},
		})
		return
	}

	JSON(w, status, models.ErrorResponse{
		Error: models.ErrorValue{
			Code: "internal_error",
			Message: err.Error(),
		},
	})
}

// Tells the client everything is fine, but there's no content to send back
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}