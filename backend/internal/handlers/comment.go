package handlers

import (
	"encoding/json"
	"net/http"
	"social-network/internal/apperror"
	"social-network/internal/comment"
	"social-network/internal/middleware"
	"social-network/internal/models"
	"social-network/internal/response"
)

type CommentHandler struct {
	commentService comment.Service
}
 
func NewCommentHandler(commentService comment.Service) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// POST /api/posts/{id}/comments
func (h *CommentHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
	postID := r.PathValue("id")
 
	var req models.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.BadInput("invalid JSON"), http.StatusBadRequest)
		return
	}
 
	c, err := h.commentService.AddComment(author.ID, postID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusCreated, c)
}

// GetComments handles GET /api/posts/{id}/comments
func (h *CommentHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	viewer := middleware.UserFromContext(r.Context())
	postID := r.PathValue("id")
 
	tree, err := h.commentService.GetCommentTree(viewer.ID, postID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, tree)
}

// DELETE /api/comments/{id}
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
	commentID := r.PathValue("id")
 
	if err := h.commentService.DeleteComment(author.ID, commentID); err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.NoContent(w)
}

// POST /api/comments/{id}/image
func (h *CommentHandler) UploadCommentImage(w http.ResponseWriter, r *http.Request) {
	author := middleware.UserFromContext(r.Context())
	commentID := r.PathValue("id")
 
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		response.Error(w, apperror.BadInput("could not parse form"), http.StatusBadRequest)
		return
	}
 
	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, apperror.BadInput("image field is required"), http.StatusBadRequest)
		return
	}
	defer file.Close()
 
	path, err := h.commentService.UploadCommentImage(author.ID, commentID, file, header)
	if err != nil {
		writeServiceError(w, err)
		return
	}
 
	response.JSON(w, http.StatusOK, map[string]string{"image_url": path})
}
