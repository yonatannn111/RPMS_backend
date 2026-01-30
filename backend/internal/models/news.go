package models

import (
	"time"

	"github.com/google/uuid"
)

type News struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	ImageURL  string    `json:"image_url"`
	VideoURL  string    `json:"video_url"`
	EditorID  uuid.UUID `json:"editor_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateNewsRequest struct {
	Title    string `json:"title" binding:"required"`
	Summary  string `json:"summary" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Category string `json:"category" binding:"required"`
	ImageURL string `json:"image_url"`
	VideoURL string `json:"video_url"`
}

type UpdateNewsRequest struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Content  string `json:"content"`
	Category string `json:"category"`
	ImageURL string `json:"image_url"`
	VideoURL string `json:"video_url"`
}
