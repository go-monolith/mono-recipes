package fileservice

import "errors"

// Sentinel errors for file service operations.
var (
	// ErrFileNotFound is returned when the requested file does not exist.
	ErrFileNotFound = errors.New("file not found")

	// ErrInvalidFileID is returned when the file ID format is invalid.
	ErrInvalidFileID = errors.New("invalid file ID format")

	// ErrInvalidFilename is returned when the filename is invalid or contains dangerous characters.
	ErrInvalidFilename = errors.New("invalid filename")
)
