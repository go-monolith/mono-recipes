package api

import "time"

// UploadResponse represents the response for file upload.
type UploadResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// FileResponse represents a file in API responses.
type FileResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url,omitempty"`
}

// ListFilesResponse represents the response for listing files.
type ListFilesResponse struct {
	Files []FileResponse `json:"files"`
	Total int            `json:"total"`
}

// DeleteResponse represents the response for file deletion.
type DeleteResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}
