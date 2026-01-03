package files

import (
	"time"
)

// UploadRequest represents a file upload request.
type UploadRequest struct {
	Name        string `json:"name"`
	Data        []byte `json:"data"`
	ContentType string `json:"content_type"`
}

// UploadResponse represents a file upload response.
type UploadResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetFileRequest represents a get file request.
type GetFileRequest struct {
	ID string `json:"id"`
}

// GetFileResponse represents a get file response.
type GetFileResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Data        []byte    `json:"data"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListFilesRequest represents a list files request.
type ListFilesRequest struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ListFilesResponse represents a list files response.
type ListFilesResponse struct {
	Files []FileMetaResponse `json:"files"`
	Total int                `json:"total"`
}

// FileMetaResponse represents file metadata in list responses.
type FileMetaResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// DeleteFileRequest represents a delete file request.
type DeleteFileRequest struct {
	ID string `json:"id"`
}

// DeleteFileResponse represents a delete file response.
type DeleteFileResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}
