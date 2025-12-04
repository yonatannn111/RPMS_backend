package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	resp, err := http.Get("http://localhost:8080/debug/users")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		fmt.Printf("Status: %d\nBody: %s\n", resp.StatusCode, string(body))
		return
	}

	var users []map[string]interface{}
	json.Unmarshal(body, &users)

	fmt.Printf("Total users in database: %d\n\n", len(users))

	if len(users) == 0 {
		fmt.Println("⚠ NO USERS FOUND IN DATABASE!")
		return
	}

	for i, u := range users {
		fmt.Printf("%d. %s (%s)\n", i+1, u["name"], u["role"])
		fmt.Printf("   Email: %s\n", u["email"])
		fmt.Printf("   ID: %s\n\n", u["id"])
	}
}
