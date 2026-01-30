package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type NewsItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	ImageURL *string `json:"image_url"`
	VideoURL *string `json:"video_url"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSSL := os.Getenv("DB_SSLMODE")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(dbUser),
		url.QueryEscape(dbPass),
		dbHost,
		dbPort,
		dbName,
		dbSSL,
	)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer pool.Close()

	rows, err := pool.Query(context.Background(), "SELECT id, title, status, image_url, video_url FROM news")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	var newsList []NewsItem
	for rows.Next() {
		var item NewsItem
		rows.Scan(&item.ID, &item.Title, &item.Status, &item.ImageURL, &item.VideoURL)
		newsList = append(newsList, item)
	}

	jsonData, _ := json.MarshalIndent(newsList, "", "  ")
	fmt.Println(string(jsonData))
}
