package models

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID             uuid.UUID `json:"id" db:"id"`
	PaperID        uuid.UUID `json:"paper_id" db:"paper_id"`
	ReviewerID     uuid.UUID `json:"reviewer_id" db:"reviewer_id"`
	Rating         int       `json:"rating" db:"rating"`
	Comments       string    `json:"comments" db:"comments"`
	Recommendation string    `json:"recommendation" db:"recommendation"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type CreateReviewRequest struct {
	PaperID        uuid.UUID `json:"paper_id" binding:"required"`
	Rating         int       `json:"rating" binding:"required,min=1,max=5"`
	Comments       string    `json:"comments"`
	Recommendation string    `json:"recommendation" binding:"required,oneof=accept minor_revision major_revision reject"`
}

type UpdateReviewRequest struct {
	Rating         int    `json:"rating" binding:"required,min=1,max=5"`
	Comments       string `json:"comments"`
	Recommendation string `json:"recommendation" binding:"required,oneof=accept minor_revision major_revision reject"`
}

type ReviewWithReviewer struct {
	Review
	ReviewerName  string `json:"reviewer_name" db:"reviewer_name"`
	ReviewerEmail string `json:"reviewer_email" db:"reviewer_email"`
	PaperTitle    string `json:"paper_title" db:"paper_title"`
}

func (r *Review) IsAccept() bool {
	return r.Recommendation == "accept"
}

func (r *Review) IsMinorRevision() bool {
	return r.Recommendation == "minor_revision"
}

func (r *Review) IsMajorRevision() bool {
	return r.Recommendation == "major_revision"
}

func (r *Review) IsReject() bool {
	return r.Recommendation == "reject"
}

func (r *Review) IsValidRating() bool {
	return r.Rating >= 1 && r.Rating <= 5
}
