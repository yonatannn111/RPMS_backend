package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		{"editor@smu.edu", "123456", "Editor User", "editor"},
		{"admin@smu.edu", "123456", "Admin User", "admin"},
		{"coordinator@smu.edu", "123456", "Coordinator User", "coordinator"},
	}

	fmt.Println("Creating SMU users...")

	for _, u := range users {
		jsonData, _ := json.Marshal(u)
		resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			fmt.Printf("❌ Failed to create %s: %v\n", u.Name, err)
			continue
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("✅ Created: %s (%s) - %s\n", u.Name, u.Role, u.Email)
			fmt.Printf("   Response: %s\n", string(body))
		} else {
			fmt.Printf("⚠️  %s (%s) - Status: %d\n", u.Name, u.Role, resp.StatusCode)
			fmt.Printf("   Response: %s\n", string(body))
		}
	}
}
