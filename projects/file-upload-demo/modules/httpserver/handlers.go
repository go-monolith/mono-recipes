package httpserver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/example/file-upload-demo/modules/fileservice"
	"github.com/gin-gonic/gin"
)

// contentTypeByExt maps file extensions to MIME types.
var contentTypeByExt = map[string]string{
	".txt":  "text/plain",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".xml":  "application/xml",
	".pdf":  "application/pdf",
	".zip":  "application/zip",
	".tar":  "application/x-tar",
	".gz":   "application/gzip",
	".gzip": "application/gzip",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

// Handlers contains HTTP request handlers for file operations.
type Handlers struct {
	fileService *fileservice.Service
}

// NewHandlers creates a new handlers instance.
func NewHandlers(fileService *fileservice.Service) *Handlers {
	return &Handlers{fileService: fileService}
}

// handleFileServiceError writes an appropriate HTTP error response for file service errors.
func handleFileServiceError(c *gin.Context, err error, operation string) {
	if errors.Is(err, fileservice.ErrFileNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	if errors.Is(err, fileservice.ErrInvalidFileID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID format"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   fmt.Sprintf("Failed to %s", operation),
		"details": err.Error(),
	})
}

// UploadFile handles file upload requests (POST /api/v1/files).
// Supports both single file and multipart form uploads.
func (h *Handlers) UploadFile(c *gin.Context) {
	// Get the uploaded file from the request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No file provided",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// Determine content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(header.Filename)
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to read file",
			"details": err.Error(),
		})
		return
	}

	// Upload the file
	result, err := h.fileService.UploadFile(c.Request.Context(), header.Filename, data, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to upload file",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UploadMultipleFiles handles multiple file uploads (POST /api/v1/files/batch).
func (h *Handlers) UploadMultipleFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid multipart form",
			"details": err.Error(),
		})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No files provided",
		})
		return
	}

	var results []fileservice.UploadResult
	var uploadErrors []string

	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("Failed to open %s: %v", header.Filename, err))
			continue
		}

		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = detectContentType(header.Filename)
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("Failed to read %s: %v", header.Filename, err))
			continue
		}

		result, err := h.fileService.UploadFile(c.Request.Context(), header.Filename, data, contentType)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("Failed to upload %s: %v", header.Filename, err))
			continue
		}

		results = append(results, *result)
	}

	response := gin.H{
		"uploaded": results,
		"count":    len(results),
	}
	if len(uploadErrors) > 0 {
		response["errors"] = uploadErrors
	}

	c.JSON(http.StatusCreated, response)
}

// ListFiles handles file listing requests (GET /api/v1/files).
func (h *Handlers) ListFiles(c *gin.Context) {
	result, err := h.fileService.ListFiles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list files",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetFile handles file download requests (GET /api/v1/files/:id).
func (h *Handlers) GetFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	data, info, err := h.fileService.GetFile(c.Request.Context(), fileID)
	if err != nil {
		handleFileServiceError(c, err, "get file")
		return
	}

	// Set response headers with sanitized filename
	safeFilename := strings.ReplaceAll(info.Name, "\"", "")
	safeFilename = strings.ReplaceAll(safeFilename, "\n", "")
	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", safeFilename))
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("X-File-ID", info.ID)
	c.Header("X-File-Digest", info.Digest)

	c.Data(http.StatusOK, info.ContentType, data)
}

// GetFileInfo handles file metadata requests (GET /api/v1/files/:id/info).
func (h *Handlers) GetFileInfo(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	info, err := h.fileService.GetFileInfo(c.Request.Context(), fileID)
	if err != nil {
		handleFileServiceError(c, err, "get file info")
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": info})
}

// DeleteFile handles file deletion requests (DELETE /api/v1/files/:id).
func (h *Handlers) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	err := h.fileService.DeleteFile(c.Request.Context(), fileID)
	if err != nil {
		handleFileServiceError(c, err, "delete file")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File deleted successfully",
		"id":      fileID,
	})
}

// HealthCheck handles health check requests (GET /health).
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "file-upload-demo",
	})
}

// detectContentType determines the content type based on file extension.
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if contentType, ok := contentTypeByExt[ext]; ok {
		return contentType
	}
	return "application/octet-stream"
}
