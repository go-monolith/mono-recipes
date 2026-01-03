package files

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	domain "github.com/example/file-upload-demo/domain/file"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// FilesModule provides file storage services using NATS JetStream Object Store.
type FilesModule struct {
	store   *JetStreamObjectStore
	service *Service
	natsURL string
	bucket  string
}

// Compile-time interface checks.
var _ mono.Module = (*FilesModule)(nil)
var _ mono.ServiceProviderModule = (*FilesModule)(nil)
var _ mono.HealthCheckableModule = (*FilesModule)(nil)

// NewModule creates a new FilesModule.
func NewModule() *FilesModule {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	bucket := os.Getenv("NATS_BUCKET")
	if bucket == "" {
		bucket = "files"
	}
	return &FilesModule{
		natsURL: natsURL,
		bucket:  bucket,
	}
}

// Name returns the module name.
func (m *FilesModule) Name() string {
	return "files"
}

// RegisterServices registers request-reply services in the service container.
func (m *FilesModule) RegisterServices(container mono.ServiceContainer) error {
	// Register upload service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"upload-file",
		json.Unmarshal,
		json.Marshal,
		m.uploadFile,
	); err != nil {
		return fmt.Errorf("failed to register upload-file service: %w", err)
	}

	// Register get-file service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"get-file",
		json.Unmarshal,
		json.Marshal,
		m.getFile,
	); err != nil {
		return fmt.Errorf("failed to register get-file service: %w", err)
	}

	// Register list-files service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"list-files",
		json.Unmarshal,
		json.Marshal,
		m.listFiles,
	); err != nil {
		return fmt.Errorf("failed to register list-files service: %w", err)
	}

	// Register delete-file service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"delete-file",
		json.Unmarshal,
		json.Marshal,
		m.deleteFile,
	); err != nil {
		return fmt.Errorf("failed to register delete-file service: %w", err)
	}

	log.Printf("[files] Registered services: upload-file, get-file, list-files, delete-file")
	return nil
}

// Start initializes the module and connects to NATS JetStream.
func (m *FilesModule) Start(ctx context.Context) error {
	var err error
	m.store, err = NewJetStreamObjectStore(m.natsURL, m.bucket)
	if err != nil {
		return fmt.Errorf("failed to create object store: %w", err)
	}

	if err := m.store.Init(ctx); err != nil {
		m.store.Close()
		return fmt.Errorf("failed to initialize object store: %w", err)
	}

	m.service = NewService(m.store)

	log.Printf("[files] Module started (NATS: %s, bucket: %s)", m.natsURL, m.bucket)
	return nil
}

// Stop shuts down the module.
func (m *FilesModule) Stop(_ context.Context) error {
	if m.store != nil {
		m.store.Close()
	}
	log.Println("[files] Module stopped")
	return nil
}

// Health returns the health status of the module.
func (m *FilesModule) Health(_ context.Context) mono.HealthStatus {
	healthy := m.store != nil && m.store.IsConnected()
	message := "connected"
	if !healthy {
		message = "disconnected"
	}
	return mono.HealthStatus{
		Healthy: healthy,
		Message: message,
		Details: map[string]any{
			"nats_url": m.natsURL,
			"bucket":   m.bucket,
		},
	}
}

// uploadFile handles the upload-file service request.
func (m *FilesModule) uploadFile(ctx context.Context, req UploadRequest, _ *mono.Msg) (UploadResponse, error) {
	meta, err := m.service.Upload(ctx, req.Name, req.Data, req.ContentType)
	if err != nil {
		return UploadResponse{}, err
	}

	return UploadResponse{
		ID:          meta.ID,
		Name:        meta.Name,
		Size:        meta.Size,
		ContentType: meta.ContentType,
		CreatedAt:   meta.CreatedAt,
	}, nil
}

// getFile handles the get-file service request.
func (m *FilesModule) getFile(ctx context.Context, req GetFileRequest, _ *mono.Msg) (GetFileResponse, error) {
	data, meta, err := m.service.Get(ctx, req.ID)
	if err != nil {
		return GetFileResponse{}, err
	}

	return GetFileResponse{
		ID:          meta.ID,
		Name:        meta.Name,
		Size:        meta.Size,
		ContentType: meta.ContentType,
		Data:        data,
		CreatedAt:   meta.CreatedAt,
	}, nil
}

// listFiles handles the list-files service request.
func (m *FilesModule) listFiles(ctx context.Context, req ListFilesRequest, _ *mono.Msg) (ListFilesResponse, error) {
	files, total, err := m.service.List(ctx, req.Limit, req.Offset)
	if err != nil {
		return ListFilesResponse{}, err
	}

	response := ListFilesResponse{
		Files: make([]FileMetaResponse, 0, len(files)),
		Total: total,
	}

	for _, f := range files {
		response.Files = append(response.Files, toFileMetaResponse(f))
	}

	return response, nil
}

// deleteFile handles the delete-file service request.
func (m *FilesModule) deleteFile(ctx context.Context, req DeleteFileRequest, _ *mono.Msg) (DeleteFileResponse, error) {
	if err := m.service.Delete(ctx, req.ID); err != nil {
		return DeleteFileResponse{Deleted: false, ID: req.ID}, err
	}

	return DeleteFileResponse{Deleted: true, ID: req.ID}, nil
}

// toFileMetaResponse converts domain FileMeta to FileMetaResponse.
func toFileMetaResponse(meta *domain.FileMeta) FileMetaResponse {
	return FileMetaResponse{
		ID:          meta.ID,
		Name:        meta.Name,
		Size:        meta.Size,
		ContentType: meta.ContentType,
		CreatedAt:   meta.CreatedAt,
	}
}
