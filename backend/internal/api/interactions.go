package api

import (
	"context"
	"fmt"
	"net/http"
	"rpms-backend/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LikePost toggles a like on a news post or event
func (s *Server) LikePost(c *gin.Context) {
	var req models.LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	ctx := c.Request.Context()

	// Check if already liked
	var existingLikeID uuid.UUID
	checkQuery := "SELECT id FROM likes WHERE user_id = $1 AND post_type = $2 AND post_id = $3"
	err = s.db.Pool.QueryRow(ctx, checkQuery, userID, req.PostType, req.PostID).Scan(&existingLikeID)

	if err == nil {
		// Already liked, so unlike
		deleteQuery := "DELETE FROM likes WHERE id = $1"
		_, err = s.db.Pool.Exec(ctx, deleteQuery, existingLikeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Unliked successfully", "liked": false})
		return
	}

	// Not liked yet, so like it
	like := models.Like{
		ID:        uuid.New(),
		UserID:    userID,
		PostType:  req.PostType,
		PostID:    req.PostID,
		CreatedAt: time.Now(),
	}

	insertQuery := `
		INSERT INTO likes (id, user_id, post_type, post_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = s.db.Pool.Exec(ctx, insertQuery, like.ID, like.UserID, like.PostType, like.PostID, like.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like"})
		return
	}

	// Create notification for coordinator if it's news or event
	go s.createEngagementNotification(req.PostType, req.PostID, userID, "like")

	c.JSON(http.StatusOK, gin.H{"message": "Liked successfully", "liked": true, "like": like})
}

// GetPostLikes gets all likes for a specific post
func (s *Server) GetPostLikes(c *gin.Context) {
	postType := c.Param("postType")
	postID := c.Param("postId")

	if postType != "news" && postType != "event" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post type"})
		return
	}

	postUUID, err := uuid.Parse(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	ctx := c.Request.Context()
	query := `
		SELECT l.id, l.user_id, l.post_type, l.post_id, l.created_at, u.name, u.avatar
		FROM likes l
		JOIN users u ON l.user_id = u.id
		WHERE l.post_type = $1 AND l.post_id = $2
		ORDER BY l.created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query, postType, postUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch likes"})
		return
	}
	defer rows.Close()

	var likes []map[string]interface{}
	for rows.Next() {
		var like models.Like
		var userName, userAvatar string
		err := rows.Scan(&like.ID, &like.UserID, &like.PostType, &like.PostID, &like.CreatedAt, &userName, &userAvatar)
		if err != nil {
			continue
		}
		likes = append(likes, map[string]interface{}{
			"id":          like.ID,
			"user_id":     like.UserID,
			"user_name":   userName,
			"user_avatar": userAvatar,
			"created_at":  like.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"likes": likes, "count": len(likes)})
}

// AddComment adds a comment to a news post or event
func (s *Server) AddComment(c *gin.Context) {
	var req models.CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	ctx := c.Request.Context()

	comment := models.Comment{
		ID:        uuid.New(),
		UserID:    userID,
		PostType:  req.PostType,
		PostID:    req.PostID,
		Content:   req.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	insertQuery := `
		INSERT INTO comments (id, user_id, post_type, post_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = s.db.Pool.Exec(ctx, insertQuery, comment.ID, comment.UserID, comment.PostType, comment.PostID, comment.Content, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}

	// Get user info for response
	var userName, userAvatar string
	userQuery := "SELECT name, avatar FROM users WHERE id = $1"
	s.db.Pool.QueryRow(ctx, userQuery, userID).Scan(&userName, &userAvatar)

	comment.UserName = userName
	comment.UserAvatar = userAvatar

	// Create notification for coordinator
	go s.createEngagementNotification(req.PostType, req.PostID, userID, "comment")

	c.JSON(http.StatusOK, gin.H{"message": "Comment added successfully", "comment": comment})
}

// GetComments gets all comments for a specific post
func (s *Server) GetComments(c *gin.Context) {
	postType := c.Param("postType")
	postID := c.Param("postId")

	if postType != "news" && postType != "event" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post type"})
		return
	}

	postUUID, err := uuid.Parse(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	ctx := c.Request.Context()
	query := `
		SELECT c.id, c.user_id, c.post_type, c.post_id, c.content, c.created_at, c.updated_at, u.name, u.avatar
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_type = $1 AND c.post_id = $2
		ORDER BY c.created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query, postType, postUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(&comment.ID, &comment.UserID, &comment.PostType, &comment.PostID, &comment.Content, &comment.CreatedAt, &comment.UpdatedAt, &comment.UserName, &comment.UserAvatar)
		if err != nil {
			continue
		}
		comments = append(comments, comment)
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments, "count": len(comments)})
}

// ShareToMessage shares a post to a chat message
func (s *Server) ShareToMessage(c *gin.Context) {
	var req models.ShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	ctx := c.Request.Context()

	// Get post title
	var postTitle string
	if req.PostType == "news" {
		s.db.Pool.QueryRow(ctx, "SELECT title FROM news WHERE id = $1", req.PostID).Scan(&postTitle)
	} else {
		s.db.Pool.QueryRow(ctx, "SELECT title FROM events WHERE id = $1", req.PostID).Scan(&postTitle)
	}

	// Create message content
	messageContent := fmt.Sprintf("Shared a %s: %s\n\n%s", req.PostType, postTitle, req.MessageText)

	// Create message
	messageID := uuid.New()
	insertMessageQuery := `
		INSERT INTO messages (id, sender_id, receiver_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = s.db.Pool.Exec(ctx, insertMessageQuery, messageID, userID, req.RecipientID, messageContent, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share"})
		return
	}

	// Record share
	share := models.Share{
		ID:        uuid.New(),
		UserID:    userID,
		PostType:  req.PostType,
		PostID:    req.PostID,
		MessageID: messageID,
		CreatedAt: time.Now(),
	}

	insertShareQuery := `
		INSERT INTO shares (id, user_id, post_type, post_id, message_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = s.db.Pool.Exec(ctx, insertShareQuery, share.ID, share.UserID, share.PostType, share.PostID, share.MessageID, share.CreatedAt)
	if err != nil {
		// Share record failed but message was sent, that's okay
		fmt.Printf("Failed to record share: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shared successfully", "message_id": messageID})
}

// GetEngagementStats gets engagement statistics for a post
func (s *Server) GetEngagementStats(c *gin.Context) {
	postType := c.Param("postType")
	postID := c.Param("postId")

	if postType != "news" && postType != "event" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post type"})
		return
	}

	postUUID, err := uuid.Parse(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	stats := models.EngagementStats{
		PostType: postType,
		PostID:   postUUID,
	}

	// Get likes count
	s.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM likes WHERE post_type = $1 AND post_id = $2", postType, postUUID).Scan(&stats.LikesCount)

	// Get comments count
	s.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM comments WHERE post_type = $1 AND post_id = $2", postType, postUUID).Scan(&stats.CommentsCount)

	// Get shares count
	s.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM shares WHERE post_type = $1 AND post_id = $2", postType, postUUID).Scan(&stats.SharesCount)

	// Check if current user liked
	if userID != nil {
		userIDStr, ok := userID.(string)
		if ok {
			userUUID, err := uuid.Parse(userIDStr)
			if err == nil {
				var likeID uuid.UUID
				err := s.db.Pool.QueryRow(ctx, "SELECT id FROM likes WHERE user_id = $1 AND post_type = $2 AND post_id = $3", userUUID, postType, postUUID).Scan(&likeID)
				stats.UserLiked = err == nil
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}

// Helper function to create engagement notifications
func (s *Server) createEngagementNotification(postType string, postID uuid.UUID, userID uuid.UUID, action string) {
	ctx := context.Background()

	// Get coordinator user ID
	var coordinatorID uuid.UUID
	err := s.db.Pool.QueryRow(ctx, "SELECT id FROM users WHERE role = 'coordinator' LIMIT 1").Scan(&coordinatorID)
	if err != nil {
		return // No coordinator found
	}

	// Get user name
	var userName string
	s.db.Pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", userID).Scan(&userName)

	// Get post title
	var postTitle string
	if postType == "news" {
		s.db.Pool.QueryRow(ctx, "SELECT title FROM news WHERE id = $1", postID).Scan(&postTitle)
	} else {
		s.db.Pool.QueryRow(ctx, "SELECT title FROM events WHERE id = $1", postID).Scan(&postTitle)
	}

	message := fmt.Sprintf("%s %sd your %s: %s", userName, action, postType, postTitle)

	// Create notification
	notificationID := uuid.New()
	insertQuery := `
		INSERT INTO notifications (id, user_id, message, is_read, created_at)
		VALUES ($1, $2, $3, false, $4)
	`
	s.db.Pool.Exec(ctx, insertQuery, notificationID, coordinatorID, message, time.Now())
}
