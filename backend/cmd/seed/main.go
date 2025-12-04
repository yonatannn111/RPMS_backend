package main

import (
	"context"
	"log"
	"time"

	"rpms-backend/internal/config"
	"rpms-backend/internal/database"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize configuration
	cfg := config.New()

	// Initialize database connection
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create users
	users := []struct {
		Email    string
		Password string
		Name     string
		Role     string
	}{
		{"author@example.com", "password123", "Alice Author", "author"},
		{"editor@example.com", "password123", "Bob Editor", "editor"},
		{"coordinator@example.com", "password123", "Charlie Coordinator", "coordinator"},
		{"admin@example.com", "password123", "David Admin", "admin"},
	}

	for _, u := range users {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)

		_, err := db.Pool.Exec(ctx, `
			INSERT INTO users (email, password_hash, name, role, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
			ON CONFLICT (email) DO NOTHING
		`, u.Email, string(hashedPassword), u.Name, u.Role, time.Now())

		if err != nil {
			log.Printf("Failed to create user %s: %v\n", u.Email, err)
		} else {
			log.Printf("User %s created (or already exists)\n", u.Email)
		}
	}

	log.Println("Seeding completed successfully!")
}
