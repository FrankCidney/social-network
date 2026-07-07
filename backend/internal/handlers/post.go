package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/post"
	"social-network/internal/response"
)

type PostHandler struct {
	postService post.Service
}

func NewPostHandler(postService post.Service) *PostHandler {
	return &PostHandler{postService: postService}
}

// POST /api/posts
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
 
	var req models.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadInput("invalid JSON"), http.StatusBadRequest)
		return
	}
 
	p, err := h.postService.CreatePost(author.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusCreated, p)
}

// GET /api/posts/{id}
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	postID := r.PathValue("id")
 
	p, err := h.postService.GetPost(viewer.ID, postID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, p)
}

// PUT /api/posts/{id}
func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
	postID := r.PathValue("id")
 
	var req models.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadInput("invalid JSON"), http.StatusBadRequest)
		return
	}
 
	if err := h.postService.UpdatePost(author.ID, postID, req); err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.NoContent(w)
}

// DELETE /api/posts/{id}
func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
	postID := r.PathValue("id")
 
	if err := h.postService.DeletePost(author.ID, postID); err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.NoContent(w)
}

// POST /api/posts/{id}/image
func (h *PostHandler) UploadPostImage(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
	postID := r.PathValue("id")
 
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		fmt.Println("header", r.Header)
		slog.Error("handlers.UploadPostImage: could not parse form", "error", err)
		response.Error(w, apperror.BadInput("could not parse form"), http.StatusBadRequest)
		return
	}
 
	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, apperror.BadInput("image field is required"), http.StatusBadRequest)
		return
	}
	defer file.Close()
 
	path, err := h.postService.UploadPostImage(author.ID, postID, file, header)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, map[string]string{"image_url": path})
}

// GET /api/feed
func (h *PostHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	limit, offset := parsePagination(r)
 
	feed, err := h.postService.GetFeed(viewer.ID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, feed)
}

// GET /api/users/{id}/posts
func (h *PostHandler) GetPostsByAuthor(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	authorID := r.PathValue("id")
	limit, offset := parsePagination(r)
 
	posts, err := h.postService.GetPostsByAuthor(viewer.ID, authorID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, posts)
}

// GET /api/groups/{id}/posts
func (h *PostHandler) GetGroupPosts(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	groupID := r.PathValue("id")
	limit, offset := parsePagination(r)
 
	posts, err := h.postService.GetGroupPosts(viewer.ID, groupID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, posts)
}
