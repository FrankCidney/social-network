package handlers

import (
	"encoding/json"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/response"
	"social-network/internal/user"
	"strconv"
)

type UserHandler struct {
	userService user.Service
}

func NewUserHandler(userService user.Service) *UserHandler {
	return &UserHandler{userService: userService}
}

// GET /api/profile/{id}
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	targetID := r.PathValue("id")
	if targetID == "" {
		targetID = viewer.ID
	}

	profile, err := h.userService.GetProfile(viewer.ID, targetID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, profile)
}

// PUT /api/profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadInput("invalid JSON"), http.StatusBadRequest)
		return
	}

	if err := h.userService.UpdateProfile(viewer.ID, req); err != nil {
		writeServiceError(w, err)
		return
	}

	response.NoContent(w)
}

// POST /api/profile/avatar
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		response.Error(w, apperror.BadInput("could not parse form"), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		response.Error(w, apperror.BadInput("avatar field is required"), http.StatusBadRequest)
		return
	}
	defer file.Close()

	path, err := h.userService.UploadAvatar(viewer.ID, file, header)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"avatar_path": path})
}

// GET /api/users/{id}/followers
func (h *UserHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	userID := r.PathValue("id")
	limit, offset := parsePagination(r)

	result, err := h.userService.GetFollowers(viewer.ID, userID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GET /api/users/{id}/following
func (h *UserHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	userID := r.PathValue("id")
	limit, offset := parsePagination(r)

	result, err := h.userService.GetFollowing(viewer.ID, userID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GET /api/users/search
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	query := r.URL.Query().Get("q")
	excludeGroupID := r.URL.Query().Get("exclude_group_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	users, err := h.userService.SearchUsers(viewer.ID, query, limit, excludeGroupID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, users)
}
