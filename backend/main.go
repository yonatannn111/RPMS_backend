package main

import (
	"log"
	"os"

	"rpms-backend/internal/api"
	"rpms-backend/internal/config"
	"rpms-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize configuration
	cfg := config.New()

	// Check if we're in demo mode (no database)
	demoMode := os.Getenv("DEMO_MODE") == "true"

	var db *database.Database
	if demoMode {
		log.Println("Running in DEMO MODE - no database connection required")
		// Create a nil database for demo mode
		db = nil
	} else {
		// Initialize database connection
		var err error
		db, err = database.NewConnection(cfg)
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()

		// Run database migrations
		if err := database.RunMigrations(db); err != nil {
			log.Fatal("Failed to run migrations:", err)
		}
	}

	// Initialize Gin router
	router := gin.Default()

	// Configure CORS
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup API routes
	api.SetupRoutes(router, db, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if demoMode {
		log.Println("🚀 Demo mode active - using mock data")
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
