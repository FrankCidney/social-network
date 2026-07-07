package models

const (
	PrivacyPublic        = "public"
	PrivacyAlmostPrivate = "almost_private"
	PrivacyPrivate       = "private"
	PrivacyGroup         = "group" // requires GroupID to be set
)

// Post represents a social media post
type Post struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	GroupID   *string   `json:"group_id,omitempty"` // Nullable
	Content   string    `json:"content,omitempty"`
	ImageURL  string    `json:"image_url,omitempty"`
	Privacy   string    `json:"privacy"` // public, almost_private, private, group
	CreatedAt string `json:"created_at"`
}

// Visibility is NOT stored here. A comment's visibility is always whatever its parent post's visibility is.
type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	ImageURL  string    `json:"image_url"`
	ParentCommentID *string `json:"parent_comment_id,omitempty"`
	CreatedAt string `json:"created_at"`
}
