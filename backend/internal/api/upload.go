package api

import (
	"fmt"
	"net/http"
	"strings"

	"rpms-backend/internal/storage"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	storage *storage.SupabaseStorage
}

func NewUploadHandler(supabaseStorage *storage.SupabaseStorage) *UploadHandler {
	return &UploadHandler{
		storage: supabaseStorage,
	}
}

// UploadFile handles file uploads for chat attachments
func (h *UploadHandler) UploadFile(c *gin.Context) {
	// Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	// Validate file size (10MB max)
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 10MB limit"})
		return
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"image/jpeg":         true,
		"image/jpg":          true,
		"image/png":          true,
		"image/gif":          true,
		"image/webp":         true,
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"text/plain": true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File type not allowed: %s", contentType)})
		return
	}

	// Upload to Supabase
	url, err := h.storage.UploadFile(file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}

	// Return file metadata
	c.JSON(http.StatusOK, gin.H{
		"url":  url,
		"name": header.Filename,
		"type": contentType,
		"size": header.Size,
	})
}

// Helper function to get file extension from content type
func getFileExtension(contentType string) string {
	parts := strings.Split(contentType, "/")
	if len(parts) == 2 {
		return "." + parts[1]
	}
	return ""
}
