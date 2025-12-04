package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	baseURL := "http://localhost:8080/api/v1/auth"

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

	fmt.Println("=== Creating and Verifying Test Users ===\n")

	createdUsers := []string{}

	for _, u := range users {
		// Try to register
		jsonData, _ := json.Marshal(u)
		resp, err := http.Post(baseURL+"/register", "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			fmt.Printf("❌ Network error for %s: %v\n", u.Name, err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("✅ Created: %s (%s)\n", u.Name, u.Role)
			createdUsers = append(createdUsers, u.Email)
		} else {
			fmt.Printf("⚠️  %s - Status %d: %s\n", u.Name, resp.StatusCode, string(body))
			// Try to login to verify it exists
			loginBody, _ := json.Marshal(map[string]string{
				"email":    u.Email,
				"password": u.Password,
			})
			loginResp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(loginBody))
			if err == nil && loginResp.StatusCode == 200 {
				fmt.Printf("   ✓ User exists (login successful)\n")
				createdUsers = append(createdUsers, u.Email)
				loginResp.Body.Close()
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Users available: %d\n\n", len(createdUsers))

	if len(createdUsers) >= 2 {
		fmt.Println("✅ Sufficient users for testing!")
		fmt.Println("\nYou can now login with:")
		for _, email := range createdUsers {
			fmt.Printf("  - %s / password123\n", email)
		}
	} else {
		fmt.Println("❌ Not enough users created!")
	}
}
