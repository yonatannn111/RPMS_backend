package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	// Login
	loginBody := map[string]string{
		"email":    "alice.author@test.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(loginBody)

	resp, err := http.Post("http://localhost:8080/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}
	defer resp.Body.Close()

	var loginResp struct {
		User struct {
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"user"`
		Token string `json:"token"`
	}

	json.NewDecoder(resp.Body).Decode(&loginResp)
	fmt.Printf("Logged in as: %s (%s)\n", loginResp.User.Name, loginResp.User.Role)
	fmt.Printf("Token: %s...\n\n", loginResp.Token[:20])

	// Get Contacts
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/v1/chat/contacts", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("GetContacts failed:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("GetContacts Response (Status %d):\n%s\n", resp.StatusCode, string(body))
}
