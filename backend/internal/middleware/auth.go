package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/auth"
	"social-network/internal/config"
	"social-network/internal/models"
	"social-network/internal/response"
)

type contextKey string

const UserContextKey contextKey = "user"

func RequireAuth(authService auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			response.Error(w, apperror.Unauthorized("authentication required"), http.StatusUnauthorized)
			return
		}

		user, err := authService.ValidateSession(cookie.Value)
		if err != nil {
			var appErr *apperror.AppError
			if errors.As(err, &appErr) && errors.Is(appErr.Err, apperror.ErrUnauthorized) {
				// Clear an invalid or stale cookie on the client side
				http.SetCookie(w, expiredCookie())
				response.Error(w, appErr, http.StatusUnauthorized)

				return
			}

			slog.Error("unhandled error reached auth middleware", "error", err)
			response.Error(w, apperror.Internal("authentication failed"), http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		panic("no user in context. check if route is missing RequireAuth")
	}
	return user
}

func expiredCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   config.SecureCookiesEnabled(),
	}
}
