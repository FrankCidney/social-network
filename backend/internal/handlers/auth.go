package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/auth"
	"social-network/internal/config"
	"social-network/internal/models"
	"social-network/internal/response"
	"time"
)

type AuthHandler struct {
	authService auth.Service
}

func NewAuthHandler(authService auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadInput("invalid JSON"), http.StatusBadRequest)
		return
	}

	authResp, err := h.authService.Register(req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	setSessionCookie(w, authResp.Token)
	response.JSON(w, http.StatusCreated, authResp)
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadInput("invalid JSON"), http.StatusBadRequest)
		return
	}

	authResp, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	setSessionCookie(w, authResp.Token)
	response.JSON(w, http.StatusOK, authResp)
}

// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		// RequireAuth already validated this exists
		slog.Error("handlers.Logout: RequireAuth middleware missing on this route", "error", err)

		response.NoContent(w)
		return
	}

	if err := h.authService.Logout(cookie.Value); err != nil {
		writeServiceError(w, err)
		return
	}

	// Clear the cookie on the client
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   config.SecureCookiesEnabled(),
	})

	response.NoContent(w)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   config.SecureCookiesEnabled(),
	})
}
