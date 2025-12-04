package models

import (
	"time"

	"github.com/google/uuid"
)

type Paper struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Abstract  string    `json:"abstract" db:"abstract"`
	Content   string    `json:"content" db:"content"`
	AuthorID  uuid.UUID `json:"author_id" db:"author_id"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreatePaperRequest struct {
	Title    string `json:"title" binding:"required,max=500"`
	Abstract string `json:"abstract"`
	Content  string `json:"content"`
}

type UpdatePaperRequest struct {
	Title    string `json:"title" binding:"required,max=500"`
	Abstract string `json:"abstract"`
	Content  string `json:"content"`
	Status   string `json:"status" binding:"oneof=draft submitted under_review approved rejected published"`
}

type PaperWithAuthor struct {
	Paper
	AuthorName  string `json:"author_name" db:"author_name"`
	AuthorEmail string `json:"author_email" db:"author_email"`
}

type PaperWithReviews struct {
	PaperWithAuthor
	Reviews []Review `json:"reviews,omitempty"`
}

func (p *Paper) IsDraft() bool {
	return p.Status == "draft"
}

func (p *Paper) IsSubmitted() bool {
	return p.Status == "submitted"
}

func (p *Paper) IsUnderReview() bool {
	return p.Status == "under_review"
}

func (p *Paper) IsApproved() bool {
	return p.Status == "approved"
}

func (p *Paper) IsRejected() bool {
	return p.Status == "rejected"
}

func (p *Paper) IsPublished() bool {
	return p.Status == "published"
}

func (p *Paper) CanEdit() bool {
	return p.IsDraft()
}

func (p *Paper) CanSubmit() bool {
	return p.IsDraft()
}

func (p *Paper) CanReview() bool {
	return p.IsSubmitted() || p.IsUnderReview()
}
