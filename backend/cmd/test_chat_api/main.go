package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	baseURL := "http://localhost:8080/api/v1"

	// 1. Login
	loginBody := map[string]string{
		"email":    "author_test@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(loginBody)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Login failed:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Login failed with status %d: %s\n", resp.StatusCode, string(body))
		return
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		fmt.Println("Failed to decode login response:", err)
		return
	}

	fmt.Println("Login successful, got token.")

	// 2. Get Contacts
	req, _ := http.NewRequest("GET", baseURL+"/chat/contacts", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("GetContacts failed:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("GetContacts Response Status: %d\n", resp.StatusCode)
	fmt.Printf("GetContacts Response Body: %s\n", string(body))
}
