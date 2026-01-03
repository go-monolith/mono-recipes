package files

import (
	"context"
	"fmt"
	"time"

	domain "github.com/example/file-upload-demo/domain/file"
	"github.com/google/uuid"
)

// Service provides file management operations.
type Service struct {
	store ObjectStore
}

// NewService creates a new file service.
func NewService(store ObjectStore) *Service {
	return &Service{store: store}
}

// Upload stores a file and returns its metadata.
func (s *Service) Upload(ctx context.Context, name string, data []byte, contentType string) (*domain.FileMeta, error) {
	if name == "" {
		return nil, fmt.Errorf("file name is required")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file data is empty")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Generate unique ID for the file
	fileID := uuid.New().String()
	storageName := fmt.Sprintf("%s/%s", fileID, name)

	// Store file in object store
	info, err := s.store.Put(ctx, storageName, data, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	now := time.Now()
	return &domain.FileMeta{
		ID:          fileID,
		Name:        name,
		Size:        int64(info.Size),
		ContentType: contentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Get retrieves a file by ID.
func (s *Service) Get(ctx context.Context, id string) ([]byte, *domain.FileMeta, error) {
	if id == "" {
		return nil, nil, fmt.Errorf("file ID is required")
	}

	// List all objects to find the one with matching ID prefix
	objects, err := s.store.List(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var storageName string
	var originalName string
	for _, obj := range objects {
		if len(obj.Name) > len(id)+1 && obj.Name[:len(id)] == id && obj.Name[len(id)] == '/' {
			storageName = obj.Name
			originalName = obj.Name[len(id)+1:]
			break
		}
	}

	if storageName == "" {
		return nil, nil, fmt.Errorf("file not found: %s", id)
	}

	// Get file data
	data, info, err := s.store.Get(ctx, storageName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	}

	return data, &domain.FileMeta{
		ID:          id,
		Name:        originalName,
		Size:        int64(info.Size),
		ContentType: info.ContentType,
		CreatedAt:   info.ModTime,
		UpdatedAt:   info.ModTime,
	}, nil
}

// List returns all files.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.FileMeta, int, error) {
	objects, err := s.store.List(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list objects: %w", err)
	}

	total := len(objects)

	// Convert to FileMeta and extract IDs from storage names
	files := make([]*domain.FileMeta, 0, len(objects))
	for _, obj := range objects {
		// Parse storage name format: "uuid/filename"
		id, name := parseStorageName(obj.Name)
		if id == "" {
			continue
		}

		files = append(files, &domain.FileMeta{
			ID:          id,
			Name:        name,
			Size:        int64(obj.Size),
			ContentType: obj.ContentType,
			CreatedAt:   obj.ModTime,
			UpdatedAt:   obj.ModTime,
		})
	}

	// Apply pagination
	if offset > len(files) {
		return []*domain.FileMeta{}, total, nil
	}
	files = files[offset:]
	if limit > 0 && limit < len(files) {
		files = files[:limit]
	}

	return files, total, nil
}

// Delete removes a file by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("file ID is required")
	}

	// List all objects to find the one with matching ID prefix
	objects, err := s.store.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	var storageName string
	for _, obj := range objects {
		if len(obj.Name) > len(id)+1 && obj.Name[:len(id)] == id && obj.Name[len(id)] == '/' {
			storageName = obj.Name
			break
		}
	}

	if storageName == "" {
		return fmt.Errorf("file not found: %s", id)
	}

	if err := s.store.Delete(ctx, storageName); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// parseStorageName extracts file ID and original name from storage name.
func parseStorageName(storageName string) (id, name string) {
	for i, c := range storageName {
		if c == '/' {
			return storageName[:i], storageName[i+1:]
		}
	}
	return "", storageName
}
