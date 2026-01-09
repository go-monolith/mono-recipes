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

const defaultContentType = "application/octet-stream"

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

// getContentType extracts the content type from headers, falling back to default.
func getContentType(headers map[string]string) string {
	if ct, ok := headers["Content-Type"]; ok {
		return ct
	}
	return defaultContentType
}

// findFileByID looks up a file by its ID prefix and returns the first match.
func (s *Service) findFileByID(fileID string) (*fsjetstream.ObjectInfo, error) {
	if err := validateFileID(fileID); err != nil {
		return nil, err
	}

	files, err := s.bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return nil, ErrFileNotFound
	}

	return &files[0], nil
}

// buildFileInfo creates a FileInfo from an ObjectInfo and file ID.
func buildFileInfo(fileID string, obj *fsjetstream.ObjectInfo) *FileInfo {
	return &FileInfo{
		ID:          fileID,
		Name:        extractOriginalFilename(obj.Name, fileID),
		Size:        int64(obj.Size),
		ContentType: getContentType(obj.Headers),
		Digest:      obj.Digest,
		CreatedAt:   obj.ModTime,
		Headers:     obj.Headers,
	}
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
	obj, err := s.findFileByID(fileID)
	if err != nil {
		return nil, nil, err
	}

	data, err := s.bucket.Get(obj.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	}

	return data, buildFileInfo(fileID, obj), nil
}

// GetFileStream retrieves a file as a stream (for large files).
func (s *Service) GetFileStream(ctx context.Context, fileID string) (io.ReadCloser, *FileInfo, error) {
	obj, err := s.findFileByID(fileID)
	if err != nil {
		return nil, nil, err
	}

	reader, _, err := s.bucket.GetReader(obj.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file stream: %w", err)
	}

	return reader, buildFileInfo(fileID, obj), nil
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
		fileID, _, found := strings.Cut(f.Name, "/")
		if !found {
			continue
		}

		fileMap[fileID] = FileInfo{
			ID:          fileID,
			Name:        extractOriginalFilename(f.Name, fileID),
			Size:        int64(f.Size),
			ContentType: getContentType(f.Headers),
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
	obj, err := s.findFileByID(fileID)
	if err != nil {
		return err
	}

	if err := s.bucket.Delete(obj.Name); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileInfo retrieves file metadata without content.
func (s *Service) GetFileInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	obj, err := s.findFileByID(fileID)
	if err != nil {
		return nil, err
	}

	return buildFileInfo(fileID, obj), nil
}
