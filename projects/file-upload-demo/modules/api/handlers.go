package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	maxFileSize   = 100 * 1024 * 1024 // 100 MB
	maxPagination = 1000
)

// setupRoutes configures all HTTP routes.
func (m *APIModule) setupRoutes() {
	// Health check endpoint
	m.router.GET("/health", m.healthHandler)

	// API v1 routes
	api := m.router.Group("/api/v1")

	// File endpoints
	fileRoutes := api.Group("/files")
	fileRoutes.POST("", m.uploadFile)
	fileRoutes.GET("", m.listFiles)
	fileRoutes.GET("/:id", m.getFile)
	fileRoutes.GET("/:id/download", m.downloadFile)
	fileRoutes.DELETE("/:id", m.deleteFile)
}

// healthHandler handles GET /health.
func (m *APIModule) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "healthy",
		Details: map[string]any{
			"module": "api",
			"port":   m.port,
		},
	})
}

// uploadFile handles POST /api/v1/files.
func (m *APIModule) uploadFile(c *gin.Context) {
	// Try multipart form first
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()

		// Check file size before reading
		if header.Size > maxFileSize {
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Error:   "file_too_large",
				Message: fmt.Sprintf("File size exceeds maximum of %d bytes", maxFileSize),
			})
			return
		}

		// Read file data with size limit
		data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "read_error",
				Message: "Failed to read file data",
			})
			return
		}

		// Double-check size after reading
		if len(data) > maxFileSize {
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Error:   "file_too_large",
				Message: fmt.Sprintf("File size exceeds maximum of %d bytes", maxFileSize),
			})
			return
		}

		// Get content type
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		// Upload via files adapter
		resp, err := m.filesAdapter.UploadFile(c.Request.Context(), header.Filename, data, contentType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "upload_failed",
				Message: "Failed to upload file",
			})
			return
		}

		c.JSON(http.StatusCreated, UploadResponse{
			ID:          resp.ID,
			Name:        resp.Name,
			Size:        resp.Size,
			ContentType: resp.ContentType,
			CreatedAt:   resp.CreatedAt,
		})
		return
	}

	// Try JSON body with base64 encoded data
	var req struct {
		Name        string `json:"name"`
		Data        string `json:"data"` // base64 encoded
		ContentType string `json:"content_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body. Use multipart/form-data with 'file' field or JSON with 'name', 'data' (base64), and 'content_type'",
		})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "File name is required",
		})
		return
	}

	if req.Data == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "File data is required",
		})
		return
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid base64 encoded data",
		})
		return
	}

	// Check file size
	if len(data) > maxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
			Error:   "file_too_large",
			Message: fmt.Sprintf("File size exceeds maximum of %d bytes", maxFileSize),
		})
		return
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// Upload via files adapter
	resp, err := m.filesAdapter.UploadFile(c.Request.Context(), req.Name, data, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "upload_failed",
			Message: "Failed to upload file",
		})
		return
	}

	c.JSON(http.StatusCreated, UploadResponse{
		ID:          resp.ID,
		Name:        resp.Name,
		Size:        resp.Size,
		ContentType: resp.ContentType,
		CreatedAt:   resp.CreatedAt,
	})
}

// getFile handles GET /api/v1/files/:id.
func (m *APIModule) getFile(c *gin.Context) {
	fileID := c.Param("id")
	if !isValidFileID(fileID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid file ID",
		})
		return
	}

	resp, err := m.filesAdapter.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "File not found",
		})
		return
	}

	c.JSON(http.StatusOK, FileResponse{
		ID:          resp.ID,
		Name:        resp.Name,
		Size:        resp.Size,
		ContentType: resp.ContentType,
		CreatedAt:   resp.CreatedAt,
		DownloadURL: fmt.Sprintf("/api/v1/files/%s/download", resp.ID),
	})
}

// downloadFile handles GET /api/v1/files/:id/download.
func (m *APIModule) downloadFile(c *gin.Context) {
	fileID := c.Param("id")
	if !isValidFileID(fileID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid file ID",
		})
		return
	}

	resp, err := m.filesAdapter.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "File not found",
		})
		return
	}

	// Set headers for file download with proper filename escaping
	escapedFilename := url.PathEscape(resp.Name)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", escapedFilename))
	c.Header("Content-Type", resp.ContentType)
	c.Header("Content-Length", strconv.FormatInt(resp.Size, 10))
	c.Data(http.StatusOK, resp.ContentType, resp.Data)
}

// listFiles handles GET /api/v1/files.
func (m *APIModule) listFiles(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Enforce pagination limits
	if limit > maxPagination {
		limit = maxPagination
	}
	if limit < 1 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	resp, err := m.filesAdapter.ListFiles(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "list_failed",
			Message: "Failed to list files",
		})
		return
	}

	fileResponses := make([]FileResponse, 0, len(resp.Files))
	for _, f := range resp.Files {
		fileResponses = append(fileResponses, FileResponse{
			ID:          f.ID,
			Name:        f.Name,
			Size:        f.Size,
			ContentType: f.ContentType,
			CreatedAt:   f.CreatedAt,
			DownloadURL: fmt.Sprintf("/api/v1/files/%s/download", f.ID),
		})
	}

	c.JSON(http.StatusOK, ListFilesResponse{
		Files: fileResponses,
		Total: resp.Total,
	})
}

// deleteFile handles DELETE /api/v1/files/:id.
func (m *APIModule) deleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if !isValidFileID(fileID) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid file ID",
		})
		return
	}

	err := m.filesAdapter.DeleteFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "File not found",
		})
		return
	}

	c.JSON(http.StatusOK, DeleteResponse{
		Deleted: true,
		ID:      fileID,
	})
}

// isValidFileID validates the file ID format.
func isValidFileID(id string) bool {
	// File IDs are UUIDs, should be 36 characters (with hyphens)
	if id == "" || len(id) > 100 {
		return false
	}
	// Basic validation - UUID format or reasonable alphanumeric ID
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
