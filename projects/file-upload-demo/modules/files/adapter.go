package files

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// FilesPort defines the interface for file operations from other modules.
type FilesPort interface {
	UploadFile(ctx context.Context, name string, data []byte, contentType string) (*UploadResponse, error)
	GetFile(ctx context.Context, id string) (*GetFileResponse, error)
	ListFiles(ctx context.Context, limit, offset int) (*ListFilesResponse, error)
	DeleteFile(ctx context.Context, id string) error
}

// filesAdapter wraps ServiceContainer for type-safe cross-module communication.
type filesAdapter struct {
	container mono.ServiceContainer
}

// NewFilesAdapter creates a new adapter for files services.
func NewFilesAdapter(container mono.ServiceContainer) FilesPort {
	if container == nil {
		panic("files adapter requires non-nil ServiceContainer")
	}
	return &filesAdapter{container: container}
}

// UploadFile uploads a file via the upload-file service.
func (a *filesAdapter) UploadFile(ctx context.Context, name string, data []byte, contentType string) (*UploadResponse, error) {
	req := UploadRequest{
		Name:        name,
		Data:        data,
		ContentType: contentType,
	}
	var resp UploadResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"upload-file",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("upload-file service call failed: %w", err)
	}
	return &resp, nil
}

// GetFile retrieves a file by ID via the get-file service.
func (a *filesAdapter) GetFile(ctx context.Context, id string) (*GetFileResponse, error) {
	req := GetFileRequest{ID: id}
	var resp GetFileResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"get-file",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("get-file service call failed: %w", err)
	}
	return &resp, nil
}

// ListFiles lists files via the list-files service.
func (a *filesAdapter) ListFiles(ctx context.Context, limit, offset int) (*ListFilesResponse, error) {
	req := ListFilesRequest{Limit: limit, Offset: offset}
	var resp ListFilesResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"list-files",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("list-files service call failed: %w", err)
	}
	return &resp, nil
}

// DeleteFile deletes a file via the delete-file service.
func (a *filesAdapter) DeleteFile(ctx context.Context, id string) error {
	req := DeleteFileRequest{ID: id}
	var resp DeleteFileResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"delete-file",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("delete-file service call failed: %w", err)
	}
	if !resp.Deleted {
		return fmt.Errorf("file not deleted: %s", id)
	}
	return nil
}
