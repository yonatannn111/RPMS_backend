package models

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Title         string    `json:"title" db:"title"`
	Description   string    `json:"description" db:"description"`
	Category      string    `json:"category" db:"category"`
	Status        string    `json:"status" db:"status"`
	ImageURL      string    `json:"image_url" db:"image_url"`
	VideoURL      string    `json:"video_url" db:"video_url"`
	Date          time.Time `json:"date" db:"date"`
	Location      string    `json:"location" db:"location"`
	CoordinatorID uuid.UUID `json:"coordinator_id" db:"coordinator_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type CreateEventRequest struct {
	Title       string    `json:"title" binding:"required,max=500"`
	Description string    `json:"description"`
	Category    string    `json:"category" binding:"required"`
	Date        time.Time `json:"date" binding:"required"`
	Location    string    `json:"location"`
	ImageURL    string    `json:"image_url"`
	VideoURL    string    `json:"video_url"`
}

type UpdateEventRequest struct {
	Title       string    `json:"title" binding:"required,max=500"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date" binding:"required"`
	Location    string    `json:"location"`
	ImageURL    string    `json:"image_url"`
	VideoURL    string    `json:"video_url"`
}

type EventWithCoordinator struct {
	Event
	CoordinatorName  string `json:"coordinator_name" db:"coordinator_name"`
	CoordinatorEmail string `json:"coordinator_email" db:"coordinator_email"`
}

func (e *Event) IsUpcoming() bool {
	return e.Date.After(time.Now())
}

func (e *Event) IsPast() bool {
	return e.Date.Before(time.Now())
}

func (e *Event) IsToday() bool {
	today := time.Now()
	return e.Date.Year() == today.Year() &&
		e.Date.Month() == today.Month() &&
		e.Date.Day() == today.Day()
}
