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
		Email    string
		Password string
		Name     string
		Role     string
	}{
		{"author_test@example.com", "password123", "Alice Author", "author"},
		{"editor_test@example.com", "password123", "Bob Editor", "editor"},
		{"coordinator_test@example.com", "password123", "Charlie Coordinator", "coordinator"},
		{"admin_test@example.com", "password123", "David Admin", "admin"},
	}

	for _, u := range users {
		jsonData, _ := json.Marshal(u)
		resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Failed to create user %s: %v\n", u.Email, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("Successfully created user: %s\n", u.Name)
		} else {
			fmt.Printf("Failed to create user %s. Status: %d\n", u.Name, resp.StatusCode)
		}
	}
}
