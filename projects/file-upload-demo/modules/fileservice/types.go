package fileservice

import "time"

// FileInfo represents metadata about a stored file.
type FileInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type"`
	Digest      string            `json:"digest"`
	CreatedAt   time.Time         `json:"created_at"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// UploadResult represents the result of a file upload operation.
type UploadResult struct {
	FileInfo  FileInfo `json:"file"`
	Message   string   `json:"message"`
	DurationMs int64   `json:"duration_ms"`
}

// ListResult represents the result of a file listing operation.
type ListResult struct {
	Files      []FileInfo `json:"files"`
	Total      int        `json:"total"`
	DurationMs int64      `json:"duration_ms"`
}

// DownloadResult represents the result of a file download (metadata only).
type DownloadResult struct {
	FileInfo   FileInfo `json:"file"`
	DurationMs int64    `json:"duration_ms"`
}
