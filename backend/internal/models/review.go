package models

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID               uuid.UUID `json:"id" db:"id"`
	PaperID          uuid.UUID `json:"paper_id" db:"paper_id"`
	ReviewerID       uuid.UUID `json:"reviewer_id" db:"reviewer_id"`
	Rating           int       `json:"rating" db:"rating"`
	ProblemStatement int       `json:"problem_statement" db:"problem_statement"`
	LiteratureReview int       `json:"literature_review" db:"literature_review"`
	Methodology      int       `json:"methodology" db:"methodology"`
	Results          int       `json:"results" db:"results"`
	Conclusion       int       `json:"conclusion" db:"conclusion"`
	Originality      int       `json:"originality" db:"originality"`
	ClarityOrg       int       `json:"clarity_organization" db:"clarity_organization"`
	Contribution     int       `json:"contribution_knowledge" db:"contribution_knowledge"`
	TechnicalQuality int       `json:"technical_quality" db:"technical_quality"`
	Comments         string    `json:"comments" db:"comments"`
	Recommendation   string    `json:"recommendation" db:"recommendation"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type CreateReviewRequest struct {
	PaperID          uuid.UUID `json:"paper_id" binding:"required"`
	ReviewerID       uuid.UUID `json:"reviewer_id"`
	Rating           int       `json:"rating" binding:"required,min=0,max=100"`
	ProblemStatement int       `json:"problem_statement" binding:"required,min=0,max=100"`
	LiteratureReview int       `json:"literature_review" binding:"required,min=0,max=100"`
	Methodology      int       `json:"methodology" binding:"required,min=0,max=100"`
	Results          int       `json:"results" binding:"required,min=0,max=100"`
	Conclusion       int       `json:"conclusion" binding:"required,min=0,max=100"`
	Originality      int       `json:"originality" binding:"required,min=0,max=100"`
	ClarityOrg       int       `json:"clarity_organization" binding:"required,min=0,max=100"`
	Contribution     int       `json:"contribution_knowledge" binding:"required,min=0,max=100"`
	TechnicalQuality int       `json:"technical_quality" binding:"required,min=0,max=100"`
	Comments         string    `json:"comments"`
	Recommendation   string    `json:"recommendation" binding:"required,oneof=accept minor_revision major_revision reject"`
}

type UpdateReviewRequest struct {
	Rating           int    `json:"rating" binding:"required,min=0,max=100"`
	ProblemStatement int    `json:"problem_statement" binding:"required,min=0,max=100"`
	LiteratureReview int    `json:"literature_review" binding:"required,min=0,max=100"`
	Methodology      int    `json:"methodology" binding:"required,min=0,max=100"`
	Results          int    `json:"results" binding:"required,min=0,max=100"`
	Conclusion       int    `json:"conclusion" binding:"required,min=0,max=100"`
	Originality      int    `json:"originality" binding:"required,min=0,max=100"`
	ClarityOrg       int    `json:"clarity_organization" binding:"required,min=0,max=100"`
	Contribution     int    `json:"contribution_knowledge" binding:"required,min=0,max=100"`
	TechnicalQuality int    `json:"technical_quality" binding:"required,min=0,max=100"`
	Comments         string `json:"comments"`
	Recommendation   string `json:"recommendation" binding:"required,oneof=accept minor_revision major_revision reject"`
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
