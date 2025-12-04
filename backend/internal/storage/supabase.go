package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

type SupabaseStorage struct {
	URL            string
	ServiceRoleKey string
	BucketName     string
}

type UploadResponse struct {
	Key string `json:"Key"`
}

func NewSupabaseStorage(url, serviceRoleKey, bucketName string) *SupabaseStorage {
	return &SupabaseStorage{
		URL:            url,
		ServiceRoleKey: serviceRoleKey,
		BucketName:     bucketName,
	}
}

// UploadFile uploads a file to Supabase Storage and returns the public URL
func (s *SupabaseStorage) UploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := uuid.New().String() + ext

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Create request to Supabase Storage
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.URL, s.BucketName, filename)
	req, err := http.NewRequest("POST", url, bytes.NewReader(fileBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+s.ServiceRoleKey)
	req.Header.Set("Content-Type", header.Header.Get("Content-Type"))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		// If JSON parsing fails, construct URL manually
		publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.URL, s.BucketName, filename)
		return publicURL, nil
	}

	// Construct public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.URL, s.BucketName, filename)
	return publicURL, nil
}

// DeleteFile deletes a file from Supabase Storage
func (s *SupabaseStorage) DeleteFile(filename string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.URL, s.BucketName, filename)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.ServiceRoleKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
