package api

import (
	"rpms-backend/internal/auth"
	"rpms-backend/internal/config"
	"rpms-backend/internal/database"
	"rpms-backend/internal/middleware"
	"rpms-backend/internal/storage"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, db *database.Database, cfg *config.Config) {
	server := NewServer(db, cfg)
	chatHandler := NewChatHandler(db)
	jwtManager := auth.NewJWTManager(cfg)

	// Initialize Supabase Storage
	supabaseStorage := storage.NewSupabaseStorage(
		cfg.Supabase.URL,
		cfg.Supabase.ServiceRoleKey,
		cfg.Supabase.Bucket,
	)
	uploadHandler := NewUploadHandler(supabaseStorage)

	// CORS middleware
	router.Use(middleware.CORSSpecific(cfg.GetCORSOrigins()))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "rpms-backend",
		})
	})

	// DEBUG ENDPOINT - REMOVE IN PRODUCTION
	router.GET("/debug/users", func(c *gin.Context) {
		rows, err := db.Pool.Query(c.Request.Context(), "SELECT id, name, email, role FROM users")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var users []map[string]interface{}
		for rows.Next() {
			var id, name, email, role string
			rows.Scan(&id, &name, &email, &role)
			users = append(users, map[string]interface{}{
				"id": id, "name": name, "email": email, "role": role,
			})
		}
		c.JSON(200, users)
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no authentication required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", server.Register)
			auth.POST("/login", server.Login)
			auth.POST("/verify", server.VerifyEmail)
			auth.POST("/resend-code", server.ResendVerificationCode)
		}

		// Public routes
		v1.GET("/events", server.GetEvents)
		v1.GET("/news", server.GetNews)

		// Protected routes (authentication required)
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtManager))
		{
			// User routes
			protected.GET("/profile", server.GetProfile)
			protected.PUT("/profile", server.UpdateProfile)
			protected.PUT("/auth/password", server.ChangePassword)
			protected.DELETE("/auth/account", server.DeleteAccount)
			protected.GET("/notifications", server.GetNotifications)
			protected.PUT("/notifications/:id/read", server.MarkNotificationRead)
			protected.POST("/notifications", server.CreateNotification)
			protected.GET("/users/admin", server.GetAdminUsers)

			papers := protected.Group("/papers")
			{
				papers.GET("", server.GetPapers)
				papers.POST("", middleware.AuthorOrAdmin(), server.CreatePaper)
				papers.PUT("/:id", middleware.AuthorOrAdmin(), server.UpdatePaper)
				papers.DELETE("/:id", middleware.AuthorOrAdmin(), server.DeletePaper)
				papers.POST("/:id/recommend", middleware.EditorOrAdmin(), server.RecommendPaperForPublication)
				papers.PUT("/:id/details", middleware.EditorOrCoordinatorOrAdmin(), server.UpdatePaperDetails)
			}

			// Review routes
			reviews := protected.Group("/reviews")
			{
				reviews.GET("", server.GetReviews)
				reviews.POST("", middleware.EditorOrAdmin(), server.CreateReview)
			}

			// Event routes
			events := protected.Group("/events")
			{
				events.POST("", middleware.CoordinatorOrAdmin(), server.CreateEvent)
				events.PUT("/:id", middleware.CoordinatorOrAdmin(), server.UpdateEvent)
				events.PUT("/:id/publish", middleware.CoordinatorOrAdmin(), server.PublishEvent)
				events.DELETE("/:id", middleware.CoordinatorOrAdmin(), server.DeleteEvent)
			}

			// News routes
			news := protected.Group("/news")
			{
				news.POST("", middleware.CoordinatorOrAdmin(), server.CreateNews)
				news.PUT("/:id", middleware.CoordinatorOrAdmin(), server.UpdateNews)
				news.PUT("/:id/publish", middleware.CoordinatorOrAdmin(), server.PublishNews)
				news.DELETE("/:id", middleware.CoordinatorOrAdmin(), server.DeleteNews)
			}

			// Chat routes
			chat := protected.Group("/chat")
			{
				chat.POST("/upload", uploadHandler.UploadFile)
				chat.POST("/send", chatHandler.SendMessage)
				chat.GET("/messages", chatHandler.GetMessages)
				chat.GET("/contacts", chatHandler.GetContacts)
				chat.GET("/unread-count", chatHandler.GetUnreadCount)
			}

			// Interaction routes (likes, comments, shares)
			interactions := protected.Group("/interactions")
			{
				interactions.POST("/like", server.LikePost)
				interactions.GET("/likes/:postType/:postId", server.GetPostLikes)
				interactions.POST("/comment", server.AddComment)
				interactions.GET("/comments/:postType/:postId", server.GetComments)
				interactions.POST("/share", server.ShareToMessage)
				interactions.GET("/stats/:postType/:postId", server.GetEngagementStats)
			}

			// Admin only routes
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				admin.GET("/stats", func(c *gin.Context) {
					// TODO: Implement admin statistics
					c.JSON(200, gin.H{"message": "Admin statistics endpoint"})
				})
				admin.POST("/users", server.AdminCreateUser)
				admin.GET("/staff", server.GetAdminStaff)
			}
		}
	}
}
