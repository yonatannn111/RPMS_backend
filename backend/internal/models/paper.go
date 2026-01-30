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
	FileUrl   string    `json:"file_url" db:"file_url"`
	AuthorID  uuid.UUID `json:"author_id" db:"author_id"`
	Status    string    `json:"status" db:"status"`
	Type      string    `json:"type" db:"type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Editor Submission Fields
	InstitutionCode         string     `json:"institution_code" db:"institution_code"`
	PublicationID           string     `json:"publication_id" db:"publication_id"`
	PublicationISCEDBand    string     `json:"publication_isced_band" db:"publication_isced_band"`
	PublicationTitleAmharic string     `json:"publication_title_amharic" db:"publication_title_amharic"`
	PublicationDate         *time.Time `json:"publication_date" db:"publication_date"`
	PublicationType         string     `json:"publication_type" db:"publication_type"`
	JournalType             string     `json:"journal_type" db:"journal_type"`
	JournalName             string     `json:"journal_name" db:"journal_name"`
	IndigenousKnowledge     bool       `json:"indigenous_knowledge" db:"indigenous_knowledge"`

	// Research Project Fields
	FiscalYear               string  `json:"fiscal_year" db:"fiscal_year"`
	AllocatedBudget          float64 `json:"allocated_budget" db:"allocated_budget"`
	ExternalBudget           float64 `json:"external_budget" db:"external_budget"`
	NRFFund                  float64 `json:"nrf_fund" db:"nrf_fund"`
	ResearchType             string  `json:"research_type" db:"research_type"`
	CompletionStatus         string  `json:"completion_status" db:"completion_status"`
	FemaleResearchers        int     `json:"female_researchers" db:"female_researchers"`
	MaleResearchers          int     `json:"male_researchers" db:"male_researchers"`
	OutsideFemaleResearchers int     `json:"outside_female_researchers" db:"outside_female_researchers"`
	OutsideMaleResearchers   int     `json:"outside_male_researchers" db:"outside_male_researchers"`
	BenefitedIndustry        string  `json:"benefited_industry" db:"benefited_industry"`
	EthicalClearance         string  `json:"ethical_clearance" db:"ethical_clearance"`
	PIName                   string  `json:"pi_name" db:"pi_name"`
	PIGender                 string  `json:"pi_gender" db:"pi_gender"`
	CoInvestigators          string  `json:"co_investigators" db:"co_investigators"`
	ProducedPrototype        string  `json:"produced_prototype" db:"produced_prototype"`
	HetrilCollaboration      string  `json:"hetril_collaboration" db:"hetril_collaboration"`
	SubmittedToIncubator     string  `json:"submitted_to_incubator" db:"submitted_to_incubator"`
}

type CreatePaperRequest struct {
	Title                   string `json:"title" binding:"required,max=500"`
	Abstract                string `json:"abstract"`
	Content                 string `json:"content"`
	FileUrl                 string `json:"file_url"`
	Type                    string `json:"type"`
	PublicationTitleAmharic string `json:"publication_title_amharic"`
	PublicationISCEDBand    string `json:"publication_isced_band"`
	PublicationType         string `json:"publication_type"`
	JournalType             string `json:"journal_type"`
	JournalName             string `json:"journal_name"`
}

type UpdatePaperRequest struct {
	Title    string `json:"title" binding:"required,max=500"`
	Abstract string `json:"abstract"`
	Content  string `json:"content"`
	FileUrl  string `json:"file_url"`
	Status   string `json:"status" binding:"oneof=draft submitted under_review approved rejected recommended_for_publication published"`

	// Editor Fields
	InstitutionCode         string    `json:"institution_code"`
	PublicationID           string    `json:"publication_id"`
	PublicationISCEDBand    string    `json:"publication_isced_band"`
	PublicationTitleAmharic string    `json:"publication_title_amharic"`
	PublicationDate         time.Time `json:"publication_date"`
	PublicationType         string    `json:"publication_type"`
	JournalType             string    `json:"journal_type"`
	JournalName             string    `json:"journal_name"`
	IndigenousKnowledge     bool      `json:"indigenous_knowledge"`

	// Research Project Fields
	FiscalYear               string  `json:"fiscal_year"`
	AllocatedBudget          float64 `json:"allocated_budget"`
	ExternalBudget           float64 `json:"external_budget"`
	NRFFund                  float64 `json:"nrf_fund"`
	ResearchType             string  `json:"research_type"`
	CompletionStatus         string  `json:"completion_status"`
	FemaleResearchers        int     `json:"female_researchers"`
	MaleResearchers          int     `json:"male_researchers"`
	OutsideFemaleResearchers int     `json:"outside_female_researchers"`
	OutsideMaleResearchers   int     `json:"outside_male_researchers"`
	BenefitedIndustry        string  `json:"benefited_industry"`
	EthicalClearance         string  `json:"ethical_clearance"`
	PIName                   string  `json:"pi_name"`
	PIGender                 string  `json:"pi_gender"`
	CoInvestigators          string  `json:"co_investigators"`
	ProducedPrototype        string  `json:"produced_prototype"`
	HetrilCollaboration      string  `json:"hetril_collaboration"`
	SubmittedToIncubator     string  `json:"submitted_to_incubator"`
}

type PaperWithAuthor struct {
	Paper
	AuthorName           string `json:"author_name" db:"author_name"`
	AuthorEmail          string `json:"author_email" db:"author_email"`
	AuthorAcademicYear   string `json:"author_academic_year" db:"author_academic_year"`
	AuthorType           string `json:"author_author_type" db:"author_author_type"`
	AuthorCategory       string `json:"author_author_category" db:"author_author_category"`
	AuthorAcademicRank   string `json:"author_academic_rank" db:"author_academic_rank"`
	AuthorQualification  string `json:"author_qualification" db:"author_qualification"`
	AuthorEmploymentType string `json:"author_employment_type" db:"author_employment_type"`
	AuthorGender         string `json:"author_gender" db:"author_gender"`
	AuthorDateOfBirth    string `json:"author_date_of_birth" db:"author_date_of_birth"`
	AuthorBio            string `json:"author_bio" db:"author_bio"`
	AuthorAvatar         string `json:"author_avatar" db:"author_avatar"`
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
