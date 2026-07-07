package models

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	DOB       string `json:"dob"`
	Nickname  string `json:"nickname"`
	AboutMe   string `json:"about_me"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	DOB       string `json:"dob"`
	Nickname  string `json:"nickname"`
	AboutMe   string `json:"about_me"`
	IsPublic  *bool  `json:"is_public"` // we use a pointer so that we can distinguish user sending "false" from user failing to provide a value
}

type CreatePostRequest struct {
	Content   string   `json:"content,omitempty"`
	Privacy   string   `json:"privacy"`              // public | almost_private | private | group
	VisibleTo []string `json:"visible_to,omitempty"` // only used when privacy == private
	GroupID   *string  `json:"group_id,omitempty"`
}

type UpdatePostRequest struct {
	Content   string   `json:"content,omitempty"`
	Privacy   string   `json:"privacy"`
	VisibleTo []string `json:"visible_to,omitempty"`
	GroupID   *string  `json:"group_id,omitempty"`
}

type CreateCommentRequest struct {
	Content string `json:"content,omitempty"`
	// ParentCommentID is omitted (nil) for a top-level comment, set for a reply.
	ParentCommentID *string `json:"parent_comment_id,omitempty"`
}

type CreateGroupRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CreateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventDate   string `json:"event_date"` // expected ISO8601
}

type InviteUserRequest struct {
	InviteeID string `json:"invitee_id"`
}

type EventRSVPRequest struct {
	Status string `json:"status"`
}
