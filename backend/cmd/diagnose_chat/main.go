package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

func main() {
	baseURL := "http://localhost:8080"

	// Test users to try
	testUsers := []struct {
		email    string
		password string
	}{
		{"author_test@example.com", "password123"},
		{"editor_test@example.com", "password123"},
		{"coordinator_test@example.com", "password123"},
		{"admin_test@example.com", "password123"},
	}

	fmt.Println("=== CHAT CONTACTS DIAGNOSTIC ===\n")

	// 1. Check debug endpoint for all users
	fmt.Println("1. Checking all users in database...")
	resp, err := http.Get(baseURL + "/debug/users")
	if err != nil {
		fmt.Printf("   ERROR: Cannot reach debug endpoint: %v\n", err)
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 200 {
			var users []map[string]interface{}
			json.Unmarshal(body, &users)
			fmt.Printf("   Found %d users:\n", len(users))
			for _, u := range users {
				fmt.Printf("   - %s (%s) - Role: %s\n", u["name"], u["email"], u["role"])
			}
		} else {
			fmt.Printf("   Status: %d, Body: %s\n", resp.StatusCode, string(body))
		}
	}

	fmt.Println("\n2. Testing login and GetContacts for each user...")

	for _, testUser := range testUsers {
		fmt.Printf("\n--- Testing: %s ---\n", testUser.email)

		// Login
		loginBody := map[string]string{
			"email":    testUser.email,
			"password": testUser.password,
		}
		jsonBody, _ := json.Marshal(loginBody)
		resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonBody))

		if err != nil {
			fmt.Printf("   Login failed: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("   Login failed (status %d): %s\n", resp.StatusCode, string(body))
			continue
		}

		var loginResp LoginResponse
		if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
			fmt.Printf("   Failed to decode login response: %v\n", err)
			continue
		}

		fmt.Printf("   ✓ Logged in as: %s (Role: %s)\n", loginResp.User.Name, loginResp.User.Role)

		// Get Contacts
		req, _ := http.NewRequest("GET", baseURL+"/api/v1/chat/contacts", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			fmt.Printf("   GetContacts failed: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("   GetContacts Status: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			var contacts []map[string]interface{}
			json.Unmarshal(body, &contacts)
			fmt.Printf("   Contacts returned: %d\n", len(contacts))
			if len(contacts) > 0 {
				fmt.Println("   Contact list:")
				for _, c := range contacts {
					fmt.Printf("     - %s (%s)\n", c["name"], c["role"])
				}
			} else {
				fmt.Println("   ⚠ NO CONTACTS RETURNED (empty array)")
			}
		} else {
			fmt.Printf("   Response: %s\n", string(body))
		}
	}

	fmt.Println("\n=== END DIAGNOSTIC ===")
}
