package fileops

import "time"

// FileSaveRequest represents a request to save a JSON file to the bucket
type FileSaveRequest struct {
	Filename string                 `json:"filename"`
	Content  map[string]interface{} `json:"content"`
}

// FileSaveResponse represents the response from a file save operation
type FileSaveResponse struct {
	FileID   string    `json:"file_id"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	Digest   string    `json:"digest"`
	SavedAt  time.Time `json:"saved_at"`
	Error    string    `json:"error,omitempty"`
}

// FileArchiveRequest represents a request to archive a file
type FileArchiveRequest struct {
	FileID string `json:"file_id"`
}
