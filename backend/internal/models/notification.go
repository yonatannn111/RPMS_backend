package models

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        int        `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Message   string     `json:"message" db:"message"`
	PaperID   *uuid.UUID `json:"paper_id" db:"paper_id"`
	IsRead    bool       `json:"is_read" db:"is_read"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
