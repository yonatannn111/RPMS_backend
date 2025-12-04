package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	baseURL := "http://localhost:8080/api/v1/auth/register"

	users := []struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}{
		{"alice.author@test.com", "password123", "Alice Author", "author"},
		{"bob.editor@test.com", "password123", "Bob Editor", "editor"},
		{"charlie.coordinator@test.com", "password123", "Charlie Coordinator", "coordinator"},
		{"david.admin@test.com", "password123", "David Admin", "admin"},
	}

	fmt.Println("Creating test users...")

	for _, u := range users {
		jsonData, _ := json.Marshal(u)
		resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			fmt.Printf("❌ Failed to create %s: %v\n", u.Name, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("✅ Created: %s (%s) - %s\n", u.Name, u.Role, u.Email)
		} else {
			fmt.Printf("⚠️  %s (%s) - Status: %d (may already exist)\n", u.Name, u.Role, resp.StatusCode)
		}
	}

	fmt.Println("\nDone! Try logging in with any of these accounts:")
	for _, u := range users {
		fmt.Printf("  - %s / password123\n", u.Email)
	}
}
