package api

import (
	"net/http"

	"rpms-backend/internal/auth"
	"rpms-backend/internal/config"
	"rpms-backend/internal/database"
	"rpms-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Server struct {
	db         *database.Database
	jwtManager *auth.JWTManager
	config     *config.Config
}

func NewServer(db *database.Database, cfg *config.Config) *Server {
	return &Server{
		db:         db,
		jwtManager: auth.NewJWTManager(cfg),
		config:     cfg,
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

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Name:         req.Name,
		Role:         req.Role,
		Avatar:       "",                       // Default
		Bio:          "",                       // Default
		Preferences:  map[string]interface{}{}, // Default
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO users (email, password_hash, name, role, avatar, bio, preferences)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, name, role, avatar, bio, preferences, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query, user.Email, user.PasswordHash, user.Name, user.Role, user.Avatar, user.Bio, user.Preferences).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.Avatar, &user.Bio, &user.Preferences, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
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

	c.JSON(http.StatusCreated, response)
}

func (s *Server) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	var user models.User

	query := `
		SELECT id, email, password_hash, name, role, avatar, bio, preferences, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := s.db.Pool.QueryRow(ctx, query, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role, &user.Avatar, &user.Bio, &user.Preferences, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
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
		SELECT p.id, p.title, p.abstract, p.content, p.author_id, p.status, p.created_at, p.updated_at,
			   u.name as author_name, u.email as author_email
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
			&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.AuthorID,
			&paper.Status, &paper.CreatedAt, &paper.UpdatedAt,
			&paper.AuthorName, &paper.AuthorEmail,
		)
		if err != nil {
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
		Title:    req.Title,
		Abstract: req.Abstract,
		Content:  req.Content,
		AuthorID: authorID,
		Status:   "draft",
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO papers (title, abstract, content, author_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, abstract, content, author_id, status, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query, paper.Title, paper.Abstract, paper.Content, paper.AuthorID, paper.Status).Scan(
		&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.AuthorID,
		&paper.Status, &paper.CreatedAt, &paper.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create paper"})
		return
	}

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
		SET title = $1, abstract = $2, content = $3, status = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, title, abstract, content, author_id, status, created_at, updated_at
	`

	var paper models.Paper
	err = s.db.Pool.QueryRow(ctx, query, req.Title, req.Abstract, req.Content, req.Status, paperID).Scan(
		&paper.ID, &paper.Title, &paper.Abstract, &paper.Content, &paper.AuthorID,
		&paper.Status, &paper.CreatedAt, &paper.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update paper"})
		return
	}

	c.JSON(http.StatusOK, paper)
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
			SELECT r.id, r.paper_id, r.reviewer_id, r.rating, r.comments, r.recommendation, r.created_at, r.updated_at,
				   reviewer.name as reviewer_name, reviewer.email as reviewer_email,
				   p.title as paper_title
			FROM reviews r
			LEFT JOIN users reviewer ON r.reviewer_id = reviewer.id
			LEFT JOIN papers p ON r.paper_id = p.id
			WHERE r.paper_id = $1
			ORDER BY r.created_at DESC
		`
		args = append(args, paperID)
	} else {
		query = `
			SELECT r.id, r.paper_id, r.reviewer_id, r.rating, r.comments, r.recommendation, r.created_at, r.updated_at,
				   reviewer.name as reviewer_name, reviewer.email as reviewer_email,
				   p.title as paper_title
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
			&review.ID, &review.PaperID, &review.ReviewerID, &review.Rating, &review.Comments,
			&review.Recommendation, &review.CreatedAt, &review.UpdatedAt,
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
		PaperID:        req.PaperID,
		ReviewerID:     reviewerID,
		Rating:         req.Rating,
		Comments:       req.Comments,
		Recommendation: req.Recommendation,
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO reviews (paper_id, reviewer_id, rating, comments, recommendation)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, paper_id, reviewer_id, rating, comments, recommendation, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query, review.PaperID, review.ReviewerID, review.Rating, review.Comments, review.Recommendation).Scan(
		&review.ID, &review.PaperID, &review.ReviewerID, &review.Rating, &review.Comments,
		&review.Recommendation, &review.CreatedAt, &review.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create review"})
		return
	}

	c.JSON(http.StatusCreated, review)
}

// Event Handlers
func (s *Server) GetEvents(c *gin.Context) {
	ctx := c.Request.Context()

	query := `
		SELECT e.id, e.title, e.description, e.date, e.location, e.coordinator_id, e.created_at, e.updated_at,
			   c.name as coordinator_name, c.email as coordinator_email
		FROM events e
		LEFT JOIN users c ON e.coordinator_id = c.id
		ORDER BY e.date ASC
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}
	defer rows.Close()

	var events []models.EventWithCoordinator
	for rows.Next() {
		var event models.EventWithCoordinator
		err := rows.Scan(
			&event.ID, &event.Title, &event.Description, &event.Date, &event.Location, &event.CoordinatorID,
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
		Date:          req.Date,
		Location:      req.Location,
		CoordinatorID: coordinatorID,
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO events (title, description, date, location, coordinator_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, date, location, coordinator_id, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query, event.Title, event.Description, event.Date, event.Location, event.CoordinatorID).Scan(
		&event.ID, &event.Title, &event.Description, &event.Date, &event.Location,
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
		SET title = $1, description = $2, date = $3, location = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, title, description, date, location, coordinator_id, created_at, updated_at
	`

	var event models.Event
	err = s.db.Pool.QueryRow(ctx, query, req.Title, req.Description, req.Date, req.Location, eventID).Scan(
		&event.ID, &event.Title, &event.Description, &event.Date, &event.Location,
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
