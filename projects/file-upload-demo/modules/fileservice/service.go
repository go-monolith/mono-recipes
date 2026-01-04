package fileservice

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
	"github.com/google/uuid"
)

// sanitizeFilename removes path separators and dangerous characters from filename.
func sanitizeFilename(filename string) string {
	// Get just the base filename without any directory components
	clean := filepath.Base(filepath.Clean(filename))
	// Remove any remaining path separators
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	// Handle edge cases
	if clean == "." || clean == ".." || clean == "" {
		return "unnamed"
	}
	return clean
}

// validateFileID validates that the given ID is a valid UUID.
func validateFileID(fileID string) error {
	if fileID == "" {
		return ErrInvalidFileID
	}
	if _, err := uuid.Parse(fileID); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidFileID, fileID)
	}
	return nil
}

// extractOriginalFilename extracts the original filename from a storage key.
func extractOriginalFilename(storageKey, fileID string) string {
	if len(fileID)+1 < len(storageKey) {
		return storageKey[len(fileID)+1:]
	}
	return storageKey
}

// Service provides file storage operations using the fs-jetstream plugin.
type Service struct {
	bucket fsjetstream.FileStoragePort
}

// NewService creates a new file service with the given storage bucket.
func NewService(bucket fsjetstream.FileStoragePort) *Service {
	return &Service{bucket: bucket}
}

// UploadFile stores a file with the given name and data.
func (s *Service) UploadFile(ctx context.Context, filename string, data []byte, contentType string) (*UploadResult, error) {
	start := time.Now()

	// Sanitize filename to prevent path traversal attacks
	safeFilename := sanitizeFilename(filename)

	// Generate unique ID for the file
	fileID := uuid.New().String()
	storageKey := fmt.Sprintf("%s/%s", fileID, safeFilename)

	// Store the file with metadata headers
	info, err := s.bucket.Put(ctx, storageKey, data,
		fsjetstream.WithDescription(fmt.Sprintf("File: %s", filename)),
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type":  contentType,
			"Original-Name": filename,
			"File-ID":       fileID,
			"Uploaded-At":   time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	fileInfo := FileInfo{
		ID:          fileID,
		Name:        safeFilename,
		Size:        int64(info.Size),
		ContentType: contentType,
		Digest:      info.Digest,
		CreatedAt:   info.ModTime,
		Headers: map[string]string{
			"Content-Type":  contentType,
			"Original-Name": safeFilename,
		},
	}

	return &UploadResult{
		FileInfo:   fileInfo,
		Message:    "File uploaded successfully",
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// UploadFileStream stores a file from a reader (for large files).
func (s *Service) UploadFileStream(ctx context.Context, filename string, reader io.Reader, contentType string) (*UploadResult, error) {
	start := time.Now()

	// Sanitize filename to prevent path traversal attacks
	safeFilename := sanitizeFilename(filename)

	// Generate unique ID for the file
	fileID := uuid.New().String()
	storageKey := fmt.Sprintf("%s/%s", fileID, safeFilename)

	// Store the file using streaming
	info, err := s.bucket.PutReader(storageKey, reader, 0,
		fsjetstream.WithDescription(fmt.Sprintf("File: %s", safeFilename)),
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type":  contentType,
			"Original-Name": safeFilename,
			"File-ID":       fileID,
			"Uploaded-At":   time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	fileInfo := FileInfo{
		ID:          fileID,
		Name:        safeFilename,
		Size:        int64(info.Size),
		ContentType: contentType,
		Digest:      info.Digest,
		CreatedAt:   info.ModTime,
		Headers: map[string]string{
			"Content-Type":  contentType,
			"Original-Name": safeFilename,
		},
	}

	return &UploadResult{
		FileInfo:   fileInfo,
		Message:    "File uploaded successfully",
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// GetFile retrieves a file by its ID.
func (s *Service) GetFile(ctx context.Context, fileID string) ([]byte, *FileInfo, error) {
	// Validate file ID format
	if err := validateFileID(fileID); err != nil {
		return nil, nil, err
	}

	// List files to find the one with matching ID prefix
	files, err := s.bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return nil, nil, ErrFileNotFound
	}

	// Get the first matching file
	fileInfo := files[0]
	data, err := s.bucket.Get(fileInfo.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	}

	contentType := "application/octet-stream"
	if ct, ok := fileInfo.Headers["Content-Type"]; ok {
		contentType = ct
	}

	info := &FileInfo{
		ID:          fileID,
		Name:        extractOriginalFilename(fileInfo.Name, fileID),
		Size:        int64(fileInfo.Size),
		ContentType: contentType,
		Digest:      fileInfo.Digest,
		CreatedAt:   fileInfo.ModTime,
		Headers:     fileInfo.Headers,
	}

	return data, info, nil
}

// GetFileStream retrieves a file as a stream (for large files).
func (s *Service) GetFileStream(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error) {
	// Validate file ID format
	if err := validateFileID(fileID); err != nil {
		return nil, nil, err
	}

	// List files to find the one with matching ID prefix
	files, err := s.bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return nil, nil, ErrFileNotFound
	}

	// Get the first matching file
	fileInfo := files[0]
	reader, _, err := s.bucket.GetReader(fileInfo.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file stream: %w", err)
	}

	contentType := "application/octet-stream"
	if ct, ok := fileInfo.Headers["Content-Type"]; ok {
		contentType = ct
	}

	info := &FileInfo{
		ID:          fileID,
		Name:        extractOriginalFilename(fileInfo.Name, fileID),
		Size:        int64(fileInfo.Size),
		ContentType: contentType,
		Digest:      fileInfo.Digest,
		CreatedAt:   fileInfo.ModTime,
		Headers:     fileInfo.Headers,
	}

	return reader, info, nil
}

// ListFiles returns all stored files.
func (s *Service) ListFiles(ctx context.Context) (*ListResult, error) {
	start := time.Now()

	files, err := s.bucket.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Convert to FileInfo list, grouping by file ID
	fileMap := make(map[string]FileInfo)
	for _, f := range files {
		// Extract file ID from storage key (format: fileID/filename)
		fileID := ""
		originalName := f.Name
		for i, c := range f.Name {
			if c == '/' {
				fileID = f.Name[:i]
				originalName = f.Name[i+1:]
				break
			}
		}

		if fileID == "" {
			continue
		}

		contentType := "application/octet-stream"
		if ct, ok := f.Headers["Content-Type"]; ok {
			contentType = ct
		}

		fileMap[fileID] = FileInfo{
			ID:          fileID,
			Name:        originalName,
			Size:        int64(f.Size),
			ContentType: contentType,
			Digest:      f.Digest,
			CreatedAt:   f.ModTime,
			Headers:     f.Headers,
		}
	}

	// Convert map to slice
	result := make([]FileInfo, 0, len(fileMap))
	for _, info := range fileMap {
		result = append(result, info)
	}

	return &ListResult{
		Files:      result,
		Total:      len(result),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// DeleteFile removes a file by its ID.
func (s *Service) DeleteFile(ctx context.Context, fileID string) error {
	// Validate file ID format
	if err := validateFileID(fileID); err != nil {
		return err
	}

	// List files to find the one with matching ID prefix
	files, err := s.bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return ErrFileNotFound
	}

	// Delete the file
	if err := s.bucket.Delete(files[0].Name); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileInfo retrieves file metadata without content.
func (s *Service) GetFileInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	// Validate file ID format
	if err := validateFileID(fileID); err != nil {
		return nil, err
	}

	// List files to find the one with matching ID prefix
	files, err := s.bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return nil, ErrFileNotFound
	}

	f := files[0]

	contentType := "application/octet-stream"
	if ct, ok := f.Headers["Content-Type"]; ok {
		contentType = ct
	}

	return &FileInfo{
		ID:          fileID,
		Name:        extractOriginalFilename(f.Name, fileID),
		Size:        int64(f.Size),
		ContentType: contentType,
		Digest:      f.Digest,
		CreatedAt:   f.ModTime,
		Headers:     f.Headers,
	}, nil
}
