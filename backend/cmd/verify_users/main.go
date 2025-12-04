package main

import (
	"context"
	"log"
	"rpms-backend/internal/config"
	"rpms-backend/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := config.New()
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	rows, err := db.Pool.Query(context.Background(), "SELECT email, role FROM users")
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	log.Println("Current users in database:")
	for rows.Next() {
		var email, role string
		if err := rows.Scan(&email, &role); err != nil {
			log.Fatal(err)
		}
		log.Printf("- %s (%s)\n", email, role)
	}
}
