package models

import "time"

type AuthResponse struct {
	User  *PublicUser `json:"user"`
	Token string      `json:"token"`
}

// PublicUser is what's returned to the client
type PublicUser struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Nickname   string `json:"nickname,omitempty"`
	AvatarPath string `json:"avatar_path,omitempty"`
	IsPublic   bool   `json:"is_public"`
}

// Profile is the full profile response (can be partial depending on visibility)
type Profile struct {
	User                *PublicUser `json:"user"`
	Email               string      `json:"email,omitempty"`
	AboutMe             string      `json:"about_me,omitempty"`
	DOB                 string      `json:"dob,omitempty"`
	FollowerCount       int         `json:"follower_count"`
	FollowingCount      int         `json:"following_count"`
	PostCount           int         `json:"post_count"`
	CanViewFullProfile  bool        `json:"can_view_full_profile"`
	IsOwnProfile        bool        `json:"is_own_profile"`
	IsFollowing         bool        `json:"is_following"`
	FollowRequestStatus string      `json:"follow_request_status,omitempty"`
}

type FollowListResponse struct {
	Users  []*PublicUser `json:"users"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type NotificationListResponse struct {
	Notifications []*Notification `json:"notifications"`
	Total         int             `json:"total"`
	Limit         int             `json:"limit"`
	Offset        int             `json:"offset"`
}

type UserSearchResult struct {
	ID                  string `json:"id"`
	FirstName           string `json:"first_name"`
	LastName            string `json:"last_name"`
	Nickname            string `json:"nickname,omitempty"`
	AvatarPath          string `json:"avatar_path,omitempty"`
	IsPublic            bool   `json:"is_public"`
	IsFollowing         bool   `json:"is_following"`
	FollowRequestStatus string `json:"follow_request_status,omitempty"`
}

type FollowRequestResponse struct {
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Status     string `json:"status"`
}

type GroupDetailResponse struct {
	ID               string      `json:"id"`
	CreatorID        string      `json:"creator_id"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	CreatedAt        time.Time   `json:"created_at"`
	Creator          *PublicUser `json:"creator"`
	IsCreator        bool        `json:"is_creator"`
	MembershipStatus string      `json:"membership_status"`
}

type PostResponse struct {
	ID        string      `json:"id"`
	Author    *PublicUser `json:"author"`
	GroupID   *string     `json:"group_id,omitempty"`
	Content   string      `json:"content,omitempty"`
	ImageURL  string      `json:"image_url,omitempty"`
	Privacy   string      `json:"privacy"`
	CreatedAt string      `json:"created_at"`
}

type CommentResponse struct {
	ID        string             `json:"id"`
	Author    *PublicUser        `json:"author"`
	Content   string             `json:"content,omitempty"`
	ImageURL  string             `json:"image_url,omitempty"`
	Depth     int                `json:"depth"`
	CreatedAt string             `json:"created_at"`
	Replies   []*CommentResponse `json:"replies"`
}

// PostListResponse is the paginated response for a feed.
type PostListResponse struct {
	Posts  []*PostResponse `json:"posts"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type ErrorValue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorValue `json:"error"`
}
