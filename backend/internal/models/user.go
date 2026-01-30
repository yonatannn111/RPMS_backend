package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	Email            string                 `json:"email" db:"email"`
	PasswordHash     string                 `json:"-" db:"password_hash"`
	Name             string                 `json:"name" db:"name"`
	Role             string                 `json:"role" db:"role"`
	Avatar           string                 `json:"avatar" db:"avatar"`
	Bio              string                 `json:"bio" db:"bio"`
	Preferences      map[string]interface{} `json:"preferences" db:"preferences"`
	IsVerified       bool                   `json:"is_verified" db:"is_verified"`
	VerificationCode string                 `json:"-" db:"verification_code"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`

	// Author Profile Fields
	AcademicYear   string `json:"academic_year" db:"academic_year"`
	AuthorType     string `json:"author_type" db:"author_type"`
	AuthorCategory string `json:"author_category" db:"author_category"`
	AcademicRank   string `json:"academic_rank" db:"academic_rank"`
	Qualification  string `json:"qualification" db:"qualification"`
	EmploymentType string `json:"employment_type" db:"employment_type"`
	Gender         string `json:"gender" db:"gender"`
	DateOfBirth    string `json:"date_of_birth" db:"date_of_birth"`
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=author editor admin coordinator"`

	// Author Profile Fields (Optional for non-authors, but we'll handle validation in handler or bind if needed)
	AcademicYear   string `json:"academic_year"`
	AuthorType     string `json:"author_type"`
	AuthorCategory string `json:"author_category"`
	AcademicRank   string `json:"academic_rank"`
	Qualification  string `json:"qualification"`
	EmploymentType string `json:"employment_type"`
	Gender         string `json:"gender"`
	DateOfBirth    string `json:"date_of_birth"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type ResendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type UpdateProfileRequest struct {
	Name        string                 `json:"name"`
	Avatar      string                 `json:"avatar"`
	Bio         string                 `json:"bio"`
	Preferences map[string]interface{} `json:"preferences"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (u *User) IsRole(role string) bool {
	return u.Role == role
}

func (u *User) IsAuthor() bool {
	return u.Role == "author"
}

func (u *User) IsEditor() bool {
	return u.Role == "editor"
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) IsCoordinator() bool {
	return u.Role == "coordinator"
}
