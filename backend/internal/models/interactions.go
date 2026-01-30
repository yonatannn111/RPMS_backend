package models

import (
	"time"

	"github.com/google/uuid"
)

// Like represents a user's like on a news post or event
type Like struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	PostType  string    `json:"post_type" db:"post_type"` // "news" or "event"
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Comment represents a user's comment on a news post or event
type Comment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	PostType  string    `json:"post_type" db:"post_type"` // "news" or "event"
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// User info for display
	UserName   string `json:"user_name,omitempty" db:"user_name"`
	UserAvatar string `json:"user_avatar,omitempty" db:"user_avatar"`
}

// Share represents a user sharing a post to messages
type Share struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	PostType  string    `json:"post_type" db:"post_type"` // "news" or "event"
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	MessageID uuid.UUID `json:"message_id" db:"message_id"` // Reference to chat message
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Request/Response Models

type LikeRequest struct {
	PostType string    `json:"post_type" binding:"required,oneof=news event"`
	PostID   uuid.UUID `json:"post_id" binding:"required"`
}

type CommentRequest struct {
	PostType string    `json:"post_type" binding:"required,oneof=news event"`
	PostID   uuid.UUID `json:"post_id" binding:"required"`
	Content  string    `json:"content" binding:"required,min=1,max=1000"`
}

type ShareRequest struct {
	PostType    string    `json:"post_type" binding:"required,oneof=news event"`
	PostID      uuid.UUID `json:"post_id" binding:"required"`
	RecipientID uuid.UUID `json:"recipient_id" binding:"required"`
	MessageText string    `json:"message_text"`
}

type EngagementStats struct {
	PostType      string    `json:"post_type"`
	PostID        uuid.UUID `json:"post_id"`
	LikesCount    int       `json:"likes_count"`
	CommentsCount int       `json:"comments_count"`
	SharesCount   int       `json:"shares_count"`
	UserLiked     bool      `json:"user_liked"` // Whether current user has liked
}
