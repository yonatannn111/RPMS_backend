package api

import (
	"net/http"
	"rpms-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Server) GetNews(c *gin.Context) {
	status := c.Query("status")

	ctx := c.Request.Context()
	query := `SELECT id, title, summary, content, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), editor_id, created_at, updated_at FROM news`

	if status != "" {
		query += ` WHERE status = $1`
	}

	query += ` ORDER BY created_at DESC`

	var rows pgx.Rows
	var err error

	if status != "" {
		rows, err = s.db.Pool.Query(ctx, query, status)
	} else {
		rows, err = s.db.Pool.Query(ctx, query)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}
	defer rows.Close()

	var newsList []models.News
	for rows.Next() {
		var news models.News
		err := rows.Scan(
			&news.ID, &news.Title, &news.Summary, &news.Content, &news.Category, &news.Status,
			&news.ImageURL, &news.VideoURL, &news.EditorID, &news.CreatedAt, &news.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan news"})
			return
		}
		newsList = append(newsList, news)
	}

	c.JSON(http.StatusOK, newsList)
}

func (s *Server) CreateNews(c *gin.Context) {
	var req models.CreateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	editorID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	news := models.News{
		Title:    req.Title,
		Summary:  req.Summary,
		Content:  req.Content,
		Category: req.Category,
		Status:   "draft",
		ImageURL: req.ImageURL,
		VideoURL: req.VideoURL,
		EditorID: editorID,
	}

	ctx := c.Request.Context()
	query := `
		INSERT INTO news (title, summary, content, category, status, image_url, video_url, editor_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, summary, content, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), editor_id, created_at, updated_at
	`

	err = s.db.Pool.QueryRow(ctx, query, news.Title, news.Summary, news.Content, news.Category, news.Status, news.ImageURL, news.VideoURL, news.EditorID).Scan(
		&news.ID, &news.Title, &news.Summary, &news.Content, &news.Category, &news.Status,
		&news.ImageURL, &news.VideoURL, &news.EditorID, &news.CreatedAt, &news.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create news"})
		return
	}

	c.JSON(http.StatusCreated, news)
}

func (s *Server) UpdateNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid news ID"})
		return
	}

	var req models.UpdateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	query := `
		UPDATE news
		SET title = $1, summary = $2, content = $3, category = $4, image_url = $5, video_url = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING id, title, summary, content, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), editor_id, created_at, updated_at
	`

	var news models.News
	err = s.db.Pool.QueryRow(ctx, query, req.Title, req.Summary, req.Content, req.Category, req.ImageURL, req.VideoURL, newsID).Scan(
		&news.ID, &news.Title, &news.Summary, &news.Content, &news.Category, &news.Status,
		&news.ImageURL, &news.VideoURL, &news.EditorID, &news.CreatedAt, &news.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update news"})
		return
	}

	c.JSON(http.StatusOK, news)
}

func (s *Server) DeleteNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid news ID"})
		return
	}

	ctx := c.Request.Context()
	query := `DELETE FROM news WHERE id = $1`

	_, err = s.db.Pool.Exec(ctx, query, newsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete news"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "News deleted successfully"})
}

func (s *Server) PublishNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid news ID"})
		return
	}

	ctx := c.Request.Context()
	query := `
		UPDATE news
		SET status = 'published', updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, summary, content, category, status, COALESCE(image_url, ''), COALESCE(video_url, ''), editor_id, created_at, updated_at
	`

	var news models.News
	err = s.db.Pool.QueryRow(ctx, query, newsID).Scan(
		&news.ID, &news.Title, &news.Summary, &news.Content, &news.Category, &news.Status,
		&news.ImageURL, &news.VideoURL, &news.EditorID, &news.CreatedAt, &news.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish news"})
		return
	}

	c.JSON(http.StatusOK, news)
}
