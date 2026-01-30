package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rpms-backend/internal/auth"
	"rpms-backend/internal/config"
	"rpms-backend/internal/database"
	"rpms-backend/internal/email"
	"rpms-backend/internal/models"
	"rpms-backend/internal/supabase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Server struct {
	db          *database.Database
	jwtManager  *auth.JWTManager
	config      *config.Config
	emailSender *email.EmailSender
	supabase    *supabase.Client
}

func NewServer(db *database.Database, cfg *config.Config) *Server {
	return &Server{
		db:          db,
		jwtManager:  auth.NewJWTManager(cfg),
		config:      cfg,
		emailSender: email.NewEmailSender(cfg),
		supabase:    supabase.NewClient(cfg),
	}
}

// Auth Handlers
// Auth Handlers
func (s *Server) Register(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password (we might still want to store it locally later, or just rely on Supabase)
	// hashedPassword, err := auth.HashPassword(req.Password)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
	// 	return
	// }

	// Prepare user metadata for Supabase
	metadata := map[string]interface{}{
		"name":            req.Name,
		"role":            "author",
		"academic_year":   req.AcademicYear,
		"author_type":     req.AuthorType,
		"author_category": req.AuthorCategory,
		"academic_rank":   req.AcademicRank,
		"qualification":   req.Qualification,
		"employment_type": req.EmploymentType,
		"gender":          req.Gender,
		"date_of_birth":   req.DateOfBirth,
	}

	// Register with Supabase
	_, err := s.supabase.SignUp(req.Email, req.Password, metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to register with Supabase: %v", err)})
		return
	}

	// We DO NOT insert into local DB yet. We wait for verification.

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email for the verification code from Supabase.",
		"email":   req.Email,
	})
}

func (s *Server) VerifyEmail(c *gin.Context) {
	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify with Supabase
	sbUser, err := s.supabase.Verify(req.Email, strings.TrimSpace(req.Code))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Supabase verification failed: %v", err)})
		return
	}

	ctx := c.Request.Context()
	var user models.User

	// Check if user exists in local DB
	query := `
		SELECT id, email, password_hash, name, role, avatar, bio, preferences, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err = s.db.Pool.QueryRow(ctx, query, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &user.Avatar, &user.Bio, &user.Preferences, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// User not found, insert them now
			meta := sbUser.UserMetadata

			// Helper to safely get string from metadata
			getString := func(key string) string {
				if v, ok := meta[key].(string); ok {
					return v
				}
				return ""
			}

			user.ID = uuid.MustParse(sbUser.ID)
			user.Email = sbUser.Email
			user.Name = getString("name")
			user.Role = getString("role")
			if user.Role == "" {
				user.Role = "author" // Default
			}
			user.Avatar = ""
			user.Bio = ""
			user.Preferences = map[string]interface{}{}
			user.IsVerified = true
			user.VerificationCode = ""

			// Extract other fields
			user.AcademicYear = getString("academic_year")
			user.AuthorType = getString("author_type")
			user.AuthorCategory = getString("author_category")
			user.AcademicRank = getString("academic_rank")
			user.Qualification = getString("qualification")
			user.EmploymentType = getString("employment_type")
			user.Gender = getString("gender")
			user.DateOfBirth = getString("date_of_birth")

			insertQuery := `
				INSERT INTO users (
					id, email, password_hash, name, role, avatar, bio, preferences, is_verified, verification_code,
					academic_year, author_type, author_category, academic_rank, qualification, employment_type, gender, date_of_birth
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
				RETURNING created_at, updated_at
			`

			// Note: password_hash is empty since we don't have it here.
			// If we want to keep it, we'd need to pass it in metadata or just rely on Supabase.
			// For now, we'll store an empty string or a placeholder.
			user.PasswordHash = ""

			err = s.db.Pool.QueryRow(ctx, insertQuery,
				user.ID, user.Email, user.PasswordHash, user.Name, user.Role, user.Avatar, user.Bio, user.Preferences, user.IsVerified, user.VerificationCode,
				user.AcademicYear, user.AuthorType, user.AuthorCategory, user.AcademicRank, user.Qualification, user.EmploymentType, user.Gender, user.DateOfBirth,
			).Scan(&user.CreatedAt, &user.UpdatedAt)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create user in local database: %v", err)})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking user"})
			return
		}
	} else {
		// User exists, just update verification status
		updateQuery := `
			UPDATE users
			SET is_verified = TRUE, verification_code = ''
			WHERE id = $1
		`
		_, err = s.db.Pool.Exec(ctx, updateQuery, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status in local database"})
			return
		}
		user.IsVerified = true
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := models.LoginResponse{
		User:  user,
		Token: token,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) ResendVerificationCode(c *gin.Context) {
	var req models.ResendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resend with Supabase
	err := s.supabase.Resend(req.Email)
	if err != nil {
		if sbErr, ok := err.(*supabase.SupabaseError); ok {
			if sbErr.StatusCode == 429 {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait a few seconds before requesting another code."})
				return
			}
			c.JSON(sbErr.StatusCode, gin.H{"error": sbErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to resend code via Supabase: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code resent successfully via Supabase"})
}

func (s *Server) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Authenticate with Supabase
	fmt.Printf("Attempting Supabase SignIn for: %s\n", req.Email)
	sbResp, err := s.supabase.SignIn(req.Email, req.Password)
	if err != nil {
		fmt.Printf("Supabase SignIn failed: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials (Supabase)"})
		return
	}
	fmt.Printf("Supabase SignIn successful. User ID: %s\n", sbResp.User.ID)

	ctx := c.Request.Context()
	var user models.User

	// Fetch user details from local DB
	query := `
		SELECT id, email, password_hash, name, role, avatar, bio, preferences, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err = s.db.Pool.QueryRow(ctx, query, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &user.Avatar, &user.Bio, &user.Preferences, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		fmt.Printf("Local DB lookup failed: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found in local database"})
		return
	}

	// Generate JWT token (local)
	token, err := s.jwtManager.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := models.LoginResponse{
		User:  user,
		Token: token,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	ctx := c.Request.Context()
	var user models.User

	query := `
		SELECT id, email, name, role, avatar, bio, preferences, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := s.db.Pool.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.Avatar, &user.Bio, &user.Preferences, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (s *Server) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	query := `
		UPDATE users
		SET name = $1, avatar = $2, bio = $3, preferences = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, email, name, role, avatar, bio, preferences, created_at, updated_at
	`

	var user models.User
	err = s.db.Pool.QueryRow(ctx, query, req.Name, req.Avatar, req.Bio, req.Preferences, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.Avatar, &user.Bio, &user.Preferences, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (s *Server) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get current password hash
	var currentHash string
	err = s.db.Pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", id).Scan(&currentHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify old password
	if !auth.CheckPassword(req.OldPassword, currentHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid old password"})
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// Update password
	_, err = s.db.Pool.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", newHash, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func (s *Server) DeleteAccount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx := c.Request.Context()
	_, err = s.db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

// Paper Handlers
func (s *Server) GetPapers(c *gin.Context) {
	ctx := c.Request.Context()

	query := `
		SELECT p.id, p.title, COALESCE(p.abstract, ''), COALESCE(p.content, ''), COALESCE(p.file_url, ''), p.author_id, p.status, COALESCE(p.type, 'Research Paper'), p.created_at, p.updated_at,
			   COALESCE(p.institution_code, ''), COALESCE(p.publication_id, ''), COALESCE(p.publication_isced_band, ''), COALESCE(p.publication_title_amharic, ''),
			   p.publication_date, COALESCE(p.publication_type, ''), COALESCE(p.journal_type, ''), COALESCE(p.journal_name, ''), COALESCE(p.indigenous_knowledge, false),
			   COALESCE(p.fiscal_year, ''), COALESCE(p.allocated_budget, 0), COALESCE(p.external_budget, 0), COALESCE(p.nrf_fund, 0),
			   COALESCE(p.research_type, ''), COALESCE(p.completion_status, ''), COALESCE(p.female_researchers, 0), COALESCE(p.male_researchers, 0),
			   COALESCE(p.outside_female_researchers, 0), COALESCE(p.outside_male_researchers, 0), COALESCE(p.benefited_industry, ''),
			   COALESCE(p.ethical_clearance, ''), COALESCE(p.pi_name, ''), COALESCE(p.pi_gender, ''), COALESCE(p.co_investigators, ''),
			   COALESCE(p.produced_prototype, ''), COALESCE(p.hetril_collaboration, ''), COALESCE(p.submitted_to_incubator, ''),
			   COALESCE(u.name, 'Unknown'), COALESCE(u.email, ''), COALESCE(u.academic_year, ''),
			   COALESCE(u.author_type, ''), COALESCE(u.author_category, ''),
			   COALESCE(u.academic_rank, ''), COALESCE(u.qualification, ''),
			   COALESCE(u.employment_type, ''), COALESCE(u.gender, ''), COALESCE(u.date_of_birth, ''),
			   COALESCE(u.bio, ''), COALESCE(u.avatar, '')
		FROM papers p
		LEFT JOIN users u ON p.author_id = u.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch papers"})
		return
	}
	defer rows.Close()

	var papers []models.PaperWithAuthor
	for rows.Next() {
		var paper models.PaperWithAuthor
		err := rows.Scan(
			&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.FileUrl, &paper.AuthorID,
			&paper.Status, &paper.Type, &paper.CreatedAt, &paper.UpdatedAt,
			&paper.InstitutionCode, &paper.PublicationID, &paper.PublicationISCEDBand, &paper.PublicationTitleAmharic,
			&paper.PublicationDate, &paper.PublicationType, &paper.JournalType, &paper.JournalName, &paper.IndigenousKnowledge,
			&paper.FiscalYear, &paper.AllocatedBudget, &paper.ExternalBudget, &paper.NRFFund,
			&paper.ResearchType, &paper.CompletionStatus, &paper.FemaleResearchers, &paper.MaleResearchers,
			&paper.OutsideFemaleResearchers, &paper.OutsideMaleResearchers, &paper.BenefitedIndustry,
			&paper.EthicalClearance, &paper.PIName, &paper.PIGender, &paper.CoInvestigators,
			&paper.ProducedPrototype, &paper.HetrilCollaboration, &paper.SubmittedToIncubator,
			&paper.AuthorName, &paper.AuthorEmail, &paper.AuthorAcademicYear,
			&paper.AuthorType, &paper.AuthorCategory, &paper.AuthorAcademicRank, &paper.AuthorQualification,
			&paper.AuthorEmploymentType, &paper.AuthorGender, &paper.AuthorDateOfBirth, &paper.AuthorBio, &paper.AuthorAvatar,
		)
		if err != nil {
			fmt.Printf("[GetPapers] Scan error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan paper"})
			return
		}
		papers = append(papers, paper)
	}

	c.JSON(http.StatusOK, papers)
}

func (s *Server) CreatePaper(c *gin.Context) {
	var req models.CreatePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	authorID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	paper := models.Paper{
		Title:                   req.Title,
		Abstract:                req.Abstract,
		Content:                 req.Content,
		FileUrl:                 req.FileUrl,
		AuthorID:                authorID,
		Status:                  "submitted",
		Type:                    req.Type,
		PublicationTitleAmharic: req.PublicationTitleAmharic,
		PublicationISCEDBand:    req.PublicationISCEDBand,
		PublicationType:         req.PublicationType,
		JournalType:             req.JournalType,
		JournalName:             req.JournalName,
	}

	if paper.Type == "" {
		paper.Type = "Research Paper" // Default
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO papers (
			title, abstract, content, file_url, author_id, status, type,
			publication_title_amharic, publication_isced_band, publication_type,
			journal_type, journal_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, title, COALESCE(abstract, ''), COALESCE(content, ''), COALESCE(file_url, ''), author_id, status, type, created_at, updated_at,
				  COALESCE(publication_title_amharic, ''), COALESCE(publication_isced_band, ''), COALESCE(publication_type, ''),
				  COALESCE(journal_type, ''), COALESCE(journal_name, '')
	`

	err = s.db.Pool.QueryRow(ctx, query,
		paper.Title, paper.Abstract, paper.Content, paper.FileUrl, paper.AuthorID, paper.Status, paper.Type,
		paper.PublicationTitleAmharic, paper.PublicationISCEDBand, paper.PublicationType,
		paper.JournalType, paper.JournalName,
	).Scan(
		&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.FileUrl, &paper.AuthorID,
		&paper.Status, &paper.Type, &paper.CreatedAt, &paper.UpdatedAt,
		&paper.PublicationTitleAmharic, &paper.PublicationISCEDBand, &paper.PublicationType,
		&paper.JournalType, &paper.JournalName,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create paper"})
		return
	}

	// Create notifications for all editors
	go func() {
		// Find all editors
		rows, err := s.db.Pool.Query(context.Background(), "SELECT id FROM users WHERE role = 'editor'")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var editorID uuid.UUID
				if err := rows.Scan(&editorID); err == nil {
					// Create notification
					s.db.Pool.Exec(context.Background(),
						"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
						editorID, "New paper submitted: "+paper.Title, paper.ID)
				}
			}
		}
	}()

	c.JSON(http.StatusCreated, paper)
}

func (s *Server) UpdatePaper(c *gin.Context) {
	paperID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid paper ID"})
		return
	}

	var req models.UpdatePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	query := `
		UPDATE papers
		SET title = $1, abstract = $2, content = $3, file_url = $4, status = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING id, title, COALESCE(abstract, ''), COALESCE(content, ''), COALESCE(file_url, ''), author_id, status, created_at, updated_at
	`

	var paper models.Paper
	err = s.db.Pool.QueryRow(ctx, query, req.Title, req.Abstract, req.Content, req.FileUrl, req.Status, paperID).Scan(
		&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.FileUrl, &paper.AuthorID,
		&paper.Status, &paper.CreatedAt, &paper.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update paper"})
		return
	}

	// If admin is publishing, approving or rejecting a recommended paper, notify the editor and author
	if req.Status == "published" || req.Status == "rejected" || req.Status == "approved" {
		go func() {
			statusText := req.Status

			// Notify all reviewers (editors)
			rows, err := s.db.Pool.Query(context.Background(),
				"SELECT reviewer_id FROM reviews WHERE paper_id = $1",
				paper.ID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var reviewerID uuid.UUID
					if err := rows.Scan(&reviewerID); err == nil {
						message := fmt.Sprintf("Admin decision: Paper '%s' has been %s", paper.Title, statusText)
						s.db.Pool.Exec(context.Background(),
							"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
							reviewerID, message, paper.ID)
					}
				}
			}

			// Also notify the author
			message := fmt.Sprintf("Your paper '%s' has been %s", paper.Title, statusText)
			s.db.Pool.Exec(context.Background(),
				"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
				paper.AuthorID, message, paper.ID)
		}()
	}

	c.JSON(http.StatusOK, paper)
}

func (s *Server) RecommendPaperForPublication(c *gin.Context) {
	paperID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid paper ID"})
		return
	}

	ctx := c.Request.Context()
	// Update paper status to recommended_for_publication
	query := `
		UPDATE papers
		SET status = 'recommended_for_publication', updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, COALESCE(abstract, ''), COALESCE(content, ''), COALESCE(file_url, ''), author_id, status, created_at, updated_at
	`

	var paper models.Paper
	err = s.db.Pool.QueryRow(ctx, query, paperID).Scan(
		&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.FileUrl, &paper.AuthorID,
		&paper.Status, &paper.CreatedAt, &paper.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to recommend paper"})
		return
	}

	// Notify all admins and the author
	go func() {
		rows, err := s.db.Pool.Query(context.Background(), "SELECT id FROM users WHERE role = 'admin'")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var adminID uuid.UUID
				if err := rows.Scan(&adminID); err == nil {
					message := fmt.Sprintf("Paper '%s' has been recommended for publication by an editor", paper.Title)
					s.db.Pool.Exec(context.Background(),
						"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
						adminID, message, paper.ID)
				}
			}
		}

		// Notify the author
		authorMessage := fmt.Sprintf("Your paper '%s' has been recommended for publication by an editor", paper.Title)
		s.db.Pool.Exec(context.Background(),
			"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
			paper.AuthorID, authorMessage, paper.ID)
	}()

	// Store editor ID for later notification (we'll add a column for this)
	// For now, we'll query reviews to find the editor
	go func() {
		s.db.Pool.Exec(context.Background(),
			"UPDATE papers SET updated_at = NOW() WHERE id = $1",
			paper.ID)
	}()

	c.JSON(http.StatusOK, paper)
}

func (s *Server) UpdatePaperDetails(c *gin.Context) {
	paperID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid paper ID"})
		return
	}

	var req models.UpdatePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Generate Publication ID if not provided and status is being set to something that implies publication or if it's just missing
	// For now, we'll generate it if it's empty.
	if req.PublicationID == "" {
		req.PublicationID, err = s.generatePublicationID(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Publication ID"})
			return
		}
	}

	query := `
		UPDATE papers
		SET institution_code = $1, publication_id = $2, publication_isced_band = $3,
			publication_title_amharic = $4, publication_date = $5, publication_type = $6,
			journal_type = $7, journal_name = $8, indigenous_knowledge = $9,
			fiscal_year = $10, allocated_budget = $11, external_budget = $12, nrf_fund = $13,
			research_type = $14, completion_status = $15, female_researchers = $16, male_researchers = $17,
			outside_female_researchers = $18, outside_male_researchers = $19, benefited_industry = $20,
			ethical_clearance = $21, pi_name = $22, pi_gender = $23, co_investigators = $24,
			produced_prototype = $25, hetril_collaboration = $26, submitted_to_incubator = $27,
			updated_at = NOW()
		WHERE id = $28
		RETURNING id, title, COALESCE(abstract, ''), COALESCE(content, ''), COALESCE(file_url, ''), author_id, status, created_at, updated_at,
				  COALESCE(institution_code, ''), COALESCE(publication_id, ''), COALESCE(publication_isced_band, ''), COALESCE(publication_title_amharic, ''),
				  publication_date, COALESCE(publication_type, ''), COALESCE(journal_type, ''), COALESCE(journal_name, ''), COALESCE(indigenous_knowledge, false),
				  COALESCE(fiscal_year, ''), COALESCE(allocated_budget, 0), COALESCE(external_budget, 0), COALESCE(nrf_fund, 0),
				  COALESCE(research_type, ''), COALESCE(completion_status, ''), COALESCE(female_researchers, 0), COALESCE(male_researchers, 0),
				  COALESCE(outside_female_researchers, 0), COALESCE(outside_male_researchers, 0), COALESCE(benefited_industry, ''),
				  COALESCE(ethical_clearance, ''), COALESCE(pi_name, ''), COALESCE(pi_gender, ''), COALESCE(co_investigators, ''),
				  COALESCE(produced_prototype, ''), COALESCE(hetril_collaboration, ''), COALESCE(submitted_to_incubator, '')
	`

	var paper models.Paper
	err = s.db.Pool.QueryRow(ctx, query,
		req.InstitutionCode, req.PublicationID, req.PublicationISCEDBand,
		req.PublicationTitleAmharic, req.PublicationDate, req.PublicationType,
		req.JournalType, req.JournalName, req.IndigenousKnowledge,
		req.FiscalYear, req.AllocatedBudget, req.ExternalBudget, req.NRFFund,
		req.ResearchType, req.CompletionStatus, req.FemaleResearchers, req.MaleResearchers,
		req.OutsideFemaleResearchers, req.OutsideMaleResearchers, req.BenefitedIndustry,
		req.EthicalClearance, req.PIName, req.PIGender, req.CoInvestigators,
		req.ProducedPrototype, req.HetrilCollaboration, req.SubmittedToIncubator,
		paperID,
	).Scan(
		&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.FileUrl, &paper.AuthorID,
		&paper.Status, &paper.CreatedAt, &paper.UpdatedAt,
		&paper.InstitutionCode, &paper.PublicationID, &paper.PublicationISCEDBand,
		&paper.PublicationTitleAmharic, &paper.PublicationDate, &paper.PublicationType,
		&paper.JournalType, &paper.JournalName, &paper.IndigenousKnowledge,
		&paper.FiscalYear, &paper.AllocatedBudget, &paper.ExternalBudget, &paper.NRFFund,
		&paper.ResearchType, &paper.CompletionStatus, &paper.FemaleResearchers, &paper.MaleResearchers,
		&paper.OutsideFemaleResearchers, &paper.OutsideMaleResearchers, &paper.BenefitedIndustry,
		&paper.EthicalClearance, &paper.PIName, &paper.PIGender, &paper.CoInvestigators,
		&paper.ProducedPrototype, &paper.HetrilCollaboration, &paper.SubmittedToIncubator,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update paper details"})
		return
	}

	// Notify Admin, Coordinator, and Author
	go func() {
		// Notify Admins
		rows, err := s.db.Pool.Query(context.Background(), "SELECT id FROM users WHERE role = 'admin'")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var adminID uuid.UUID
				if err := rows.Scan(&adminID); err == nil {
					message := fmt.Sprintf("Paper details updated for '%s' by Editor", paper.Title)
					s.db.Pool.Exec(context.Background(),
						"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
						adminID, message, paper.ID)
				}
			}
		}

		// Notify Coordinators
		rowsCoord, err := s.db.Pool.Query(context.Background(), "SELECT id FROM users WHERE role = 'coordinator'")
		if err == nil {
			defer rowsCoord.Close()
			for rowsCoord.Next() {
				var coordID uuid.UUID
				if err := rowsCoord.Scan(&coordID); err == nil {
					message := fmt.Sprintf("Paper details updated for '%s' by Editor. Please validate.", paper.Title)
					s.db.Pool.Exec(context.Background(),
						"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
						coordID, message, paper.ID)
				}
			}
		}

		// Notify the author
		authorMessage := fmt.Sprintf("Publication details for your paper '%s' have been updated by an editor", paper.Title)
		s.db.Pool.Exec(context.Background(),
			"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
			paper.AuthorID, authorMessage, paper.ID)
	}()

	c.JSON(http.StatusOK, paper)
}

func (s *Server) generatePublicationID(ctx context.Context) (string, error) {
	// Format: SMU_P201817001
	// We need to find the last ID and increment it.
	// Assuming the format is fixed and numeric part is at the end.

	var lastID string
	err := s.db.Pool.QueryRow(ctx, "SELECT publication_id FROM papers WHERE publication_id LIKE 'SMU_P%' ORDER BY publication_id DESC LIMIT 1").Scan(&lastID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "SMU_P201817001", nil
		}
		// If it's NULL (which it might be for existing papers), we might get here or Scan might fail depending on driver.
		// pgx Scan returns error if NULL is scanned into string without NullString? No, Scan handles it if we use *string or NullString.
		// But here we scan into string. If NULL, it errors.
		// Let's handle it safely.
		return "SMU_P201817001", nil // Default start if error or no rows
	}

	if lastID == "" {
		return "SMU_P201817001", nil
	}

	// Parse the number
	// SMU_P201817001 -> 201817001
	if len(lastID) < 6 {
		return "SMU_P201817001", nil
	}

	prefix := "SMU_P"
	numStr := lastID[len(prefix):]
	var num int
	_, err = fmt.Sscanf(numStr, "%d", &num)
	if err != nil {
		return "SMU_P201817001", nil // Fallback
	}

	newNum := num + 1
	return fmt.Sprintf("%s%d", prefix, newNum), nil
}

func (s *Server) DeletePaper(c *gin.Context) {
	paperID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid paper ID"})
		return
	}

	ctx := c.Request.Context()
	query := `DELETE FROM papers WHERE id = $1`

	_, err = s.db.Pool.Exec(ctx, query, paperID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete paper"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper deleted successfully"})
}

// Review Handlers
func (s *Server) GetReviews(c *gin.Context) {
	ctx := c.Request.Context()

	paperID := c.Query("paper_id")
	var query string
	var args []interface{}

	if paperID != "" {
		query = `
			SELECT r.id, r.paper_id, r.reviewer_id, r.rating, 
				   COALESCE(r.problem_statement, 0), COALESCE(r.literature_review, 0), 
				   COALESCE(r.methodology, 0), COALESCE(r.results, 0), COALESCE(r.conclusion, 0),
				   COALESCE(r.originality, 0), COALESCE(r.clarity_organization, 0),
				   COALESCE(r.contribution_knowledge, 0), COALESCE(r.technical_quality, 0),
				   COALESCE(r.comments, ''), r.recommendation, r.created_at, r.updated_at,
				   COALESCE(reviewer.name, 'Unknown'), COALESCE(reviewer.email, ''),
				   COALESCE(p.title, 'Unknown Paper')
			FROM reviews r
			LEFT JOIN users reviewer ON r.reviewer_id = reviewer.id
			LEFT JOIN papers p ON r.paper_id = p.id
			WHERE r.paper_id = $1
			ORDER BY r.created_at DESC
		`
		args = append(args, paperID)
	} else {
		query = `
			SELECT r.id, r.paper_id, r.reviewer_id, r.rating, 
				   COALESCE(r.problem_statement, 0), COALESCE(r.literature_review, 0), 
				   COALESCE(r.methodology, 0), COALESCE(r.results, 0), COALESCE(r.conclusion, 0),
				   COALESCE(r.originality, 0), COALESCE(r.clarity_organization, 0),
				   COALESCE(r.contribution_knowledge, 0), COALESCE(r.technical_quality, 0),
				   COALESCE(r.comments, ''), r.recommendation, r.created_at, r.updated_at,
				   COALESCE(reviewer.name, 'Unknown'), COALESCE(reviewer.email, ''),
				   COALESCE(p.title, 'Unknown Paper')
			FROM reviews r
			LEFT JOIN users reviewer ON r.reviewer_id = reviewer.id
			LEFT JOIN papers p ON r.paper_id = p.id
			ORDER BY r.created_at DESC
		`
	}

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}
	defer rows.Close()

	var reviews []models.ReviewWithReviewer
	for rows.Next() {
		var review models.ReviewWithReviewer
		err := rows.Scan(
			&review.ID, &review.PaperID, &review.ReviewerID, &review.Rating,
			&review.ProblemStatement, &review.LiteratureReview, &review.Methodology,
			&review.Results, &review.Conclusion, &review.Originality, &review.ClarityOrg,
			&review.Contribution, &review.TechnicalQuality,
			&review.Comments, &review.Recommendation, &review.CreatedAt, &review.UpdatedAt,
			&review.ReviewerName, &review.ReviewerEmail, &review.PaperTitle,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan review"})
			return
		}
		reviews = append(reviews, review)
	}

	c.JSON(http.StatusOK, reviews)
}

func (s *Server) CreateReview(c *gin.Context) {
	var req models.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	reviewerID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	review := models.Review{
		PaperID:          req.PaperID,
		ReviewerID:       reviewerID,
		Rating:           req.Rating,
		ProblemStatement: req.ProblemStatement,
		LiteratureReview: req.LiteratureReview,
		Methodology:      req.Methodology,
		Results:          req.Results,
		Conclusion:       req.Conclusion,
		Originality:      req.Originality,
		ClarityOrg:       req.ClarityOrg,
		Contribution:     req.Contribution,
		TechnicalQuality: req.TechnicalQuality,
		Comments:         req.Comments,
		Recommendation:   req.Recommendation,
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO reviews (paper_id, reviewer_id, rating, problem_statement, literature_review, methodology, results, conclusion, originality, clarity_organization, contribution_knowledge, technical_quality, comments, recommendation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, paper_id, reviewer_id, rating, problem_statement, literature_review, methodology, results, conclusion, originality, clarity_organization, contribution_knowledge, technical_quality, comments, recommendation, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query,
		review.PaperID, review.ReviewerID, review.Rating,
		review.ProblemStatement, review.LiteratureReview, review.Methodology,
		review.Results, review.Conclusion, review.Originality, review.ClarityOrg,
		review.Contribution, review.TechnicalQuality,
		review.Comments, review.Recommendation).Scan(
		&review.ID, &review.PaperID, &review.ReviewerID, &review.Rating,
		&review.ProblemStatement, &review.LiteratureReview, &review.Methodology,
		&review.Results, &review.Conclusion, &review.Originality, &review.ClarityOrg,
		&review.Contribution, &review.TechnicalQuality,
		&review.Comments, &review.Recommendation, &review.CreatedAt, &review.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create review"})
		return
	}

	// Send notification to paper author
	go func() {
		// Get paper details to find author
		var authorID uuid.UUID
		var paperTitle string
		err := s.db.Pool.QueryRow(context.Background(),
			"SELECT author_id, title FROM papers WHERE id = $1",
			review.PaperID).Scan(&authorID, &paperTitle)

		if err == nil {
			// Create notification message with review details
			message := fmt.Sprintf("Your paper '%s' has been reviewed. Rating: %d/5, Recommendation: %s",
				paperTitle, review.Rating, review.Recommendation)

			s.db.Pool.Exec(context.Background(),
				"INSERT INTO notifications (user_id, message, paper_id) VALUES ($1, $2, $3)",
				authorID, message, review.PaperID)
		}
	}()

	c.JSON(http.StatusCreated, review)
}

// Event Handlers
func (s *Server) GetEvents(c *gin.Context) {
	ctx := c.Request.Context()
	status := c.Query("status")

	query := `
		SELECT e.id, e.title, e.description, e.category, e.status, COALESCE(e.image_url, ''), COALESCE(e.video_url, ''), e.date, e.location, e.coordinator_id, e.created_at, e.updated_at,
			   c.name as coordinator_name, c.email as coordinator_email
		FROM events e
		LEFT JOIN users c ON e.coordinator_id = c.id
	`

	var args []interface{}
	if status != "" {
		query += " WHERE e.status = $1"
		args = append(args, status)
	}

	query += " ORDER BY e.date ASC"

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}
	defer rows.Close()

	var events []models.EventWithCoordinator
	for rows.Next() {
		var event models.EventWithCoordinator
		err := rows.Scan(
			&event.ID, &event.Title, &event.Description, &event.Category, &event.Status, &event.ImageURL, &event.VideoURL, &event.Date, &event.Location, &event.CoordinatorID,
			&event.CreatedAt, &event.UpdatedAt, &event.CoordinatorName, &event.CoordinatorEmail,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan event"})
			return
		}
		events = append(events, event)
	}

	c.JSON(http.StatusOK, events)
}

func (s *Server) PublishEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	ctx := c.Request.Context()
	query := `
		UPDATE events
		SET status = 'published', updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, description, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), date, location, coordinator_id, created_at, updated_at
	`

	var event models.Event
	err = s.db.Pool.QueryRow(ctx, query, eventID).Scan(
		&event.ID, &event.Title, &event.Description, &event.Category, &event.Status, &event.ImageURL, &event.VideoURL, &event.Date, &event.Location,
		&event.CoordinatorID, &event.CreatedAt, &event.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish event"})
		return
	}

	c.JSON(http.StatusOK, event)
}

func (s *Server) CreateEvent(c *gin.Context) {
	var req models.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	coordinatorID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	event := models.Event{
		Title:         req.Title,
		Description:   req.Description,
		Category:      req.Category,
		Status:        "draft",
		ImageURL:      req.ImageURL,
		VideoURL:      req.VideoURL,
		Date:          req.Date,
		Location:      req.Location,
		CoordinatorID: coordinatorID,
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO events (title, description, category, status, image_url, video_url, date, location, coordinator_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), date, location, coordinator_id, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query, event.Title, event.Description, event.Category, event.Status, event.ImageURL, event.VideoURL, event.Date, event.Location, event.CoordinatorID).Scan(
		&event.ID, &event.Title, &event.Description, &event.Category, &event.Status, &event.ImageURL, &event.VideoURL, &event.Date, &event.Location,
		&event.CoordinatorID, &event.CreatedAt, &event.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (s *Server) UpdateEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var req models.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	query := `
		UPDATE events
		SET title = $1, description = $2, category = $3, date = $4, location = $5, image_url = $6, video_url = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING id, title, description, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), date, location, coordinator_id, created_at, updated_at
	`

	var event models.Event
	err = s.db.Pool.QueryRow(ctx, query, req.Title, req.Description, req.Category, req.Date, req.Location, req.ImageURL, req.VideoURL, eventID).Scan(
		&event.ID, &event.Title, &event.Description, &event.Category, &event.Status, &event.ImageURL, &event.VideoURL, &event.Date, &event.Location,
		&event.CoordinatorID, &event.CreatedAt, &event.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	c.JSON(http.StatusOK, event)
}

func (s *Server) DeleteEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	ctx := c.Request.Context()
	query := `DELETE FROM events WHERE id = $1`

	_, err = s.db.Pool.Exec(ctx, query, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
}

// GetAdminUsers returns admin users (for editor to contact admin)
func (s *Server) GetAdminUsers(c *gin.Context) {
	ctx := c.Request.Context()

	query := `
		SELECT id, email, name, role
		FROM users
		WHERE role = 'admin'
		LIMIT 1
	`

	var adminUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}

	err := s.db.Pool.QueryRow(ctx, query).Scan(
		&adminUser.ID, &adminUser.Email, &adminUser.Name, &adminUser.Role,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No admin user found"})
		return
	}

	c.JSON(http.StatusOK, adminUser)
}

// CreateNotification creates a notification for a user
func (s *Server) CreateNotification(c *gin.Context) {
	var req struct {
		UserID  string  `json:"user_id" binding:"required"`
		Message string  `json:"message" binding:"required"`
		PaperID *string `json:"paper_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO notifications (user_id, message, paper_id)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, message, is_read, created_at, paper_id
	`

	var notification models.Notification
	err := s.db.Pool.QueryRow(ctx, query, req.UserID, req.Message, req.PaperID).Scan(
		&notification.ID, &notification.UserID, &notification.Message,
		&notification.IsRead, &notification.CreatedAt, &notification.PaperID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (s *Server) GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx := c.Request.Context()
	query := `
		SELECT id, user_id, message, paper_id, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Message,
			&notification.PaperID, &notification.IsRead, &notification.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan notification"})
			return
		}
		notifications = append(notifications, notification)
	}

	c.JSON(http.StatusOK, notifications)
}

func (s *Server) MarkNotificationRead(c *gin.Context) {
	notificationID := c.Param("id")
	fmt.Printf("[Backend] MarkNotificationRead called for ID: %s\n", notificationID)

	id, err := strconv.Atoi(notificationID)
	if err != nil {
		fmt.Printf("[Backend] Error parsing notification ID '%s': %v\n", notificationID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	ctx := c.Request.Context()

	// Log the current state before update
	var currentIsRead bool
	err = s.db.Pool.QueryRow(ctx, "SELECT is_read FROM notifications WHERE id = $1", id).Scan(&currentIsRead)
	if err != nil {
		fmt.Printf("[Backend] Error checking current status of notification %d: %v\n", id, err)
	} else {
		fmt.Printf("[Backend] Notification %d current is_read: %v\n", id, currentIsRead)
	}

	query := `
		UPDATE notifications
		SET is_read = true
		WHERE id = $1
		RETURNING id, user_id, message, paper_id, is_read, created_at
	`

	var notification models.Notification
	err = s.db.Pool.QueryRow(ctx, query, id).Scan(
		&notification.ID, &notification.UserID, &notification.Message,
		&notification.PaperID, &notification.IsRead, &notification.CreatedAt,
	)

	if err != nil {
		fmt.Printf("[Backend] Error updating notification %d: %v\n", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read: " + err.Error()})
		return
	}

	fmt.Printf("[Backend] Notification %d marked as read successfully. New status: %v\n", id, notification.IsRead)
	c.JSON(http.StatusOK, notification)
}

// Admin User Management Handlers

func (s *Server) AdminCreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role (must be editor or coordinator)
	if req.Role != "editor" && req.Role != "coordinator" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Only editor and coordinator can be created by admin."})
		return
	}

	// Create user in Supabase (confirmed)
	metadata := map[string]interface{}{
		"name": req.Name,
		"role": req.Role,
	}

	sbUser, err := s.supabase.AdminCreateUser(req.Email, req.Password, metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create user in Supabase: %v", err)})
		return
	}

	ctx := c.Request.Context()
	user := models.User{
		ID:          uuid.MustParse(sbUser.ID),
		Email:       sbUser.Email,
		Name:        req.Name,
		Role:        req.Role,
		IsVerified:  true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Preferences: map[string]interface{}{},
	}

	// Insert into local DB
	query := `
		INSERT INTO users (id, email, password_hash, name, role, is_verified, created_at, updated_at, preferences)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	// We assume simple password hash storage isn't needed locally if we rely on Supabase,
	// but we can store a placeholder or hash it if we want local fallback.
	// For now, empty string.
	_, err = s.db.Pool.Exec(ctx, query,
		user.ID, user.Email, "", user.Name, user.Role, user.IsVerified, user.CreatedAt, user.UpdatedAt, user.Preferences,
	)

	if err != nil {
		// Try to delete from Supabase if local insert fails?
		// For now just error out.
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create user in local DB: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (s *Server) GetAdminStaff(c *gin.Context) {
	ctx := c.Request.Context()
	query := `
		SELECT id, email, name, role, created_at, is_verified
		FROM users
		WHERE role IN ('editor', 'coordinator')
		ORDER BY created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff"})
		return
	}
	defer rows.Close()

	var staff []gin.H
	for rows.Next() {
		var id uuid.UUID
		var email, name, role string
		var createdAt time.Time
		var isVerified bool

		if err := rows.Scan(&id, &email, &name, &role, &createdAt, &isVerified); err != nil {
			continue
		}

		staff = append(staff, gin.H{
			"id":          id,
			"email":       email,
			"name":        name,
			"role":        role,
			"created_at":  createdAt,
			"is_verified": isVerified,
		})
	}

	c.JSON(http.StatusOK, staff)
}
