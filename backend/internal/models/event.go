package models

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Title         string    `json:"title" db:"title"`
	Description   string    `json:"description" db:"description"`
	Date          time.Time `json:"date" db:"date"`
	Location      string    `json:"location" db:"location"`
	CoordinatorID uuid.UUID `json:"coordinator_id" db:"coordinator_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type CreateEventRequest struct {
	Title       string    `json:"title" binding:"required,max=500"`
	Description string    `json:"description"`
	Date        time.Time `json:"date" binding:"required"`
	Location    string    `json:"location"`
}

type UpdateEventRequest struct {
	Title       string    `json:"title" binding:"required,max=500"`
	Description string    `json:"description"`
	Date        time.Time `json:"date" binding:"required"`
	Location    string    `json:"location"`
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
