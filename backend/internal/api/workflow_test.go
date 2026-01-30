package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rpms-backend/internal/api"
	"rpms-backend/internal/config"
	"rpms-backend/internal/database"
	"rpms-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setupTestServer(t *testing.T) (*gin.Engine, *database.Database) {
	// Load .env from root backend directory
	if err := godotenv.Load("../../.env"); err != nil { // Assuming running from internal/api
		if err := godotenv.Load(".env"); err != nil {
			// Try absolute path if needed, or just log
			log.Println("Could not load .env")
		}
	}

	cfg := config.New()
	// Force test DB if possible, or use the dev DB but be careful
	// For now we use the configured DB.

	db, err := database.NewConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations to ensure DB schema is up to date
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	api.SetupRoutes(router, db, cfg)

	return router, db
}

func TestPublicationWorkflow(t *testing.T) {
	router, db := setupTestServer(t)
	defer db.Close()

	// 1. Register Author
	authorEmail := fmt.Sprintf("author_%d@test.com", time.Now().UnixNano())
	authorPassword := "password123"
	authorName := "Test Author"

	registerReq := models.CreateUserRequest{
		Name:           authorName,
		Email:          authorEmail,
		Password:       authorPassword,
		Role:           "author",
		AcademicYear:   "2024",
		AuthorType:     "Academic Staff",
		AuthorCategory: "Researcher",
		AcademicRank:   "Lecturer",
		Qualification:  "PhD",
		EmploymentType: "Full Time",
		Gender:         "Male",
	}

	body, _ := json.Marshal(registerReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	fmt.Println("Registering author...")
	router.ServeHTTP(w, req)
	fmt.Printf("Register response: %d - %s\n", w.Code, w.Body.String())

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to register author: %d - %s", w.Code, w.Body.String())
	}

	// 2. Login Author
	loginReq := map[string]string{
		"email":    authorEmail,
		"password": authorPassword,
	}
	body, _ = json.Marshal(loginReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to login author: %d - %s", w.Code, w.Body.String())
	}

	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	authorToken := loginResp["token"].(string)

	// 3. Create Paper (Draft)
	paperTitle := "Test Paper for Workflow"
	createPaperReq := models.CreatePaperRequest{
		Title:    paperTitle,
		Abstract: "This is a test abstract.",
		Content:  "Test content.",
	}
	body, _ = json.Marshal(createPaperReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/papers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authorToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create paper: %d - %s", w.Code, w.Body.String())
	}

	var paperResp models.Paper
	json.Unmarshal(w.Body.Bytes(), &paperResp)
	paperID := paperResp.ID

	// 4. Submit Paper (Status -> submitted)
	updateStatusReq := map[string]string{
		"title":  paperTitle,
		"status": "submitted",
	}
	body, _ = json.Marshal(updateStatusReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/papers/%s", paperID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authorToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to submit paper: %d - %s", w.Code, w.Body.String())
	}

	// 5. Create Editor
	editorEmail := fmt.Sprintf("editor_%d@test.com", time.Now().UnixNano())
	editorPassword := "password123"
	insertTestUser(t, router, db, editorEmail, editorPassword, "editor")

	// Login Editor
	loginReq = map[string]string{
		"email":    editorEmail,
		"password": editorPassword,
	}
	body, _ = json.Marshal(loginReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to login editor: %d - %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	editorToken := loginResp["token"].(string)

	// 6. Editor Updates Paper Details
	pubDate, _ := time.Parse("2006-01-02", "2024-01-01")
	updateDetailsReq := models.UpdatePaperRequest{
		InstitutionCode:         "SMU",
		PublicationISCEDBand:    "Band 1",
		PublicationTitleAmharic: "Test Amharic Title",
		PublicationDate:         pubDate,
		PublicationType:         "Journal Article",
		JournalType:             "International",
		JournalName:             "Test Journal",
		IndigenousKnowledge:     true,
		Status:                  "submitted", // Status might be required or optional, keeping it same
		Title:                   paperTitle,
	}
	body, _ = json.Marshal(updateDetailsReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/papers/%s/details", paperID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+editorToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update paper details by editor: %d - %s", w.Code, w.Body.String())
	}

	var updatedPaper models.Paper
	json.Unmarshal(w.Body.Bytes(), &updatedPaper)

	if updatedPaper.PublicationID == "" {
		t.Error("PublicationID should have been generated")
	}
	if updatedPaper.InstitutionCode != "SMU" {
		t.Error("InstitutionCode mismatch")
	}

	// 7. Coordinator Validation
	coordEmail := fmt.Sprintf("coord_%d@test.com", time.Now().UnixNano())
	coordPassword := "password123"
	insertTestUser(t, router, db, coordEmail, coordPassword, "coordinator")

	// Login Coordinator
	loginReq = map[string]string{
		"email":    coordEmail,
		"password": coordPassword,
	}
	body, _ = json.Marshal(loginReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to login coordinator: %d - %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	coordToken := loginResp["token"].(string)

	// Coordinator updates details (Validation)
	updateDetailsReq.PublicationISCEDBand = "Band 2" // Change something
	updateDetailsReq.Title = paperTitle
	body, _ = json.Marshal(updateDetailsReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/papers/%s/details", paperID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+coordToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to validate paper details by coordinator: %d - %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &updatedPaper)
	if updatedPaper.PublicationISCEDBand != "Band 2" {
		t.Error("Coordinator update failed to persist")
	}

	// Cleanup
	db.Pool.Exec(context.Background(), "DELETE FROM users WHERE email IN ($1, $2, $3)", authorEmail, editorEmail, coordEmail)
	db.Pool.Exec(context.Background(), "DELETE FROM papers WHERE id = $1", paperID)
}

func insertTestUser(t *testing.T, router *gin.Engine, db *database.Database, email, password, role string) string {
	// 1. Register as Author first (default)
	registerReq := models.CreateUserRequest{
		Name:           "Test " + role,
		Email:          email,
		Password:       password,
		Role:           "author",
		AcademicYear:   "2024",
		AuthorType:     "Academic Staff",
		AuthorCategory: "Researcher",
		AcademicRank:   "Lecturer",
		Qualification:  "PhD",
		EmploymentType: "Full Time",
		Gender:         "Male",
	}

	body, _ := json.Marshal(registerReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to register user %s: %d - %s", email, w.Code, w.Body.String())
	}

	// 2. Get User ID from DB
	var userID string
	err := db.Pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to find user %s in DB: %v", email, err)
	}

	// 3. Update Role if needed
	if role != "author" {
		_, err = db.Pool.Exec(context.Background(), "UPDATE users SET role = $1 WHERE id = $2", role, userID)
		if err != nil {
			t.Fatalf("Failed to update role for user %s: %v", email, err)
		}
	}

	return userID
}
