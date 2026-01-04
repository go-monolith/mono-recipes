// Example: Using fs-jetstream Plugin
//
// This example demonstrates:
// - Configuring and registering the fs-jetstream plugin
// - Storing and retrieving files (Put, Get)
// - Streaming large files (PutReader, GetReader)
// - Adding metadata to files
// - Listing and filtering files
// - Document management use case
// - Temporary upload handling use case

package fsjetstreamusecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
)

// ============================================================
// Document Module - Document Storage Use Case
// ============================================================

type DocumentModule struct {
	storage *fsjetstream.PluginModule
	docs    fsjetstream.FileStoragePort
}

var (
	_ mono.Module          = (*DocumentModule)(nil)
	_ mono.UsePluginModule = (*DocumentModule)(nil)
)

func NewDocumentModule() *DocumentModule {
	return &DocumentModule{}
}

func (m *DocumentModule) Name() string { return "documents" }

func (m *DocumentModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "storage" {
		m.storage = plugin.(*fsjetstream.PluginModule)
	}
}

func (m *DocumentModule) Start(ctx context.Context) error {
	if m.storage == nil {
		return fmt.Errorf("required plugin 'storage' not registered")
	}

	m.docs = m.storage.Bucket("documents")
	if m.docs == nil {
		return fmt.Errorf("bucket 'documents' not found")
	}

	slog.Info("Document module started")
	return nil
}

func (m *DocumentModule) Stop(ctx context.Context) error {
	slog.Info("Document module stopped")
	return nil
}

// StoreDocument stores a document with metadata
func (m *DocumentModule) StoreDocument(ctx context.Context, name string, data []byte, contentType, author string) (*fsjetstream.ObjectInfo, error) {
	info, err := m.docs.Put(ctx, name, data,
		fsjetstream.WithDescription(fmt.Sprintf("Document: %s", name)),
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type": contentType,
			"Author":       author,
			"Created":      time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store document: %w", err)
	}

	slog.Info("Document stored",
		"name", info.Name,
		"size", info.Size,
		"digest", info.Digest)

	return info, nil
}

// GetDocument retrieves a document
func (m *DocumentModule) GetDocument(name string) ([]byte, error) {
	return m.docs.Get(name)
}

// GetDocumentInfo retrieves document metadata without content
func (m *DocumentModule) GetDocumentInfo(name string) (*fsjetstream.ObjectInfo, error) {
	return m.docs.Stat(name)
}

// ListDocuments lists all documents
func (m *DocumentModule) ListDocuments() ([]fsjetstream.ObjectInfo, error) {
	return m.docs.List()
}

// ListDocumentsByPrefix lists documents with a specific prefix
func (m *DocumentModule) ListDocumentsByPrefix(prefix string) ([]fsjetstream.ObjectInfo, error) {
	return m.docs.List(fsjetstream.WithPrefix(prefix))
}

// DeleteDocument removes a document
func (m *DocumentModule) DeleteDocument(name string) error {
	return m.docs.Delete(name)
}

// ============================================================
// Upload Module - Temporary Upload Handling Use Case
// ============================================================

type UploadModule struct {
	storage *fsjetstream.PluginModule
	uploads fsjetstream.FileStoragePort
}

var (
	_ mono.Module          = (*UploadModule)(nil)
	_ mono.UsePluginModule = (*UploadModule)(nil)
)

func NewUploadModule() *UploadModule {
	return &UploadModule{}
}

func (m *UploadModule) Name() string { return "uploads" }

func (m *UploadModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "storage" {
		m.storage = plugin.(*fsjetstream.PluginModule)
	}
}

func (m *UploadModule) Start(ctx context.Context) error {
	if m.storage == nil {
		return fmt.Errorf("required plugin 'storage' not registered")
	}

	m.uploads = m.storage.Bucket("uploads")
	if m.uploads == nil {
		return fmt.Errorf("bucket 'uploads' not found")
	}

	slog.Info("Upload module started")
	return nil
}

func (m *UploadModule) Stop(ctx context.Context) error {
	slog.Info("Upload module stopped")
	return nil
}

// UploadFile uploads a file from bytes (for small files)
func (m *UploadModule) UploadFile(ctx context.Context, filename string, data []byte) (*fsjetstream.ObjectInfo, error) {
	uploadKey := fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), filename)

	info, err := m.uploads.Put(ctx, uploadKey, data,
		fsjetstream.WithHeaders(map[string]string{
			"Original-Name": filename,
			"Upload-Time":   time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	slog.Info("File uploaded", "key", uploadKey, "size", info.Size)
	return info, nil
}

// UploadLargeFile uploads a file from a reader (for large files)
func (m *UploadModule) UploadLargeFile(filename string, reader io.Reader) (*fsjetstream.ObjectInfo, error) {
	uploadKey := fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), filename)

	info, err := m.uploads.PutReader(uploadKey, reader, 0, // 0 = use bucket's default TTL
		fsjetstream.WithHeaders(map[string]string{
			"Original-Name": filename,
			"Upload-Time":   time.Now().Format(time.RFC3339),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload large file: %w", err)
	}

	slog.Info("Large file uploaded", "key", uploadKey, "size", info.Size)
	return info, nil
}

// DownloadFile downloads a file to memory (for small files)
func (m *UploadModule) DownloadFile(key string) ([]byte, error) {
	return m.uploads.Get(key)
}

// StreamFile downloads a file as a stream (for large files)
func (m *UploadModule) StreamFile(key string) (io.ReadCloser, *fsjetstream.ObjectInfo, error) {
	return m.uploads.GetReader(key)
}

// ListUploads lists all uploaded files
func (m *UploadModule) ListUploads() ([]fsjetstream.ObjectInfo, error) {
	return m.uploads.List()
}

// ============================================================
// Media Module - Multiple Buckets Use Case
// ============================================================

type MediaModule struct {
	storage    *fsjetstream.PluginModule
	images     fsjetstream.FileStoragePort
	thumbnails fsjetstream.FileStoragePort
}

var (
	_ mono.Module          = (*MediaModule)(nil)
	_ mono.UsePluginModule = (*MediaModule)(nil)
)

func NewMediaModule() *MediaModule {
	return &MediaModule{}
}

func (m *MediaModule) Name() string { return "media" }

func (m *MediaModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "storage" {
		m.storage = plugin.(*fsjetstream.PluginModule)
	}
}

func (m *MediaModule) Start(ctx context.Context) error {
	if m.storage == nil {
		return fmt.Errorf("required plugin 'storage' not registered")
	}

	m.images = m.storage.Bucket("images")
	if m.images == nil {
		return fmt.Errorf("bucket 'images' not found")
	}

	m.thumbnails = m.storage.Bucket("thumbnails")
	if m.thumbnails == nil {
		return fmt.Errorf("bucket 'thumbnails' not found")
	}

	slog.Info("Media module started with 2 buckets")
	return nil
}

func (m *MediaModule) Stop(ctx context.Context) error {
	slog.Info("Media module stopped")
	return nil
}

// StoreImage stores an image and its thumbnail
func (m *MediaModule) StoreImage(ctx context.Context, name string, imageData, thumbnailData []byte) error {
	// Store original image
	_, err := m.images.Put(ctx, name, imageData,
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type": "image/jpeg",
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to store image: %w", err)
	}

	// Store thumbnail
	thumbName := "thumb_" + name
	_, err = m.thumbnails.Put(ctx, thumbName, thumbnailData,
		fsjetstream.WithHeaders(map[string]string{
			"Content-Type": "image/jpeg",
			"Original":     name,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to store thumbnail: %w", err)
	}

	slog.Info("Image stored with thumbnail", "name", name)
	return nil
}

// GetImage retrieves an image
func (m *MediaModule) GetImage(name string) ([]byte, error) {
	return m.images.Get(name)
}

// GetThumbnail retrieves a thumbnail
func (m *MediaModule) GetThumbnail(name string) ([]byte, error) {
	return m.thumbnails.Get("thumb_" + name)
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework fs-jetstream Plugin Example ===")
	fmt.Println("Demonstrates: Document storage, File uploads, Multiple buckets")
	fmt.Println()

	// Create temp directory for JetStream storage
	jsDir := "/tmp/mono-fs-example"

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
		mono.WithJetStreamStorageDir(jsDir),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create fs-jetstream plugin with multiple buckets
	storage, err := fsjetstream.New(fsjetstream.Config{
		Buckets: []fsjetstream.BucketConfig{
			{
				Name:        "documents",
				Description: "Permanent document storage",
				MaxBytes:    500 * 1024 * 1024, // 500MB
				Storage:     fsjetstream.FileStorage,
				Compression: true,
			},
			{
				Name:        "uploads",
				Description: "Temporary file uploads",
				TTL:         24 * time.Hour, // Auto-expire after 24 hours
				Storage:     fsjetstream.MemoryStorage,
			},
			{
				Name:        "images",
				Description: "Original images",
				MaxBytes:    1024 * 1024 * 1024, // 1GB
				Storage:     fsjetstream.FileStorage,
			},
			{
				Name:        "thumbnails",
				Description: "Image thumbnails",
				MaxBytes:    100 * 1024 * 1024, // 100MB
				Storage:     fsjetstream.MemoryStorage,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create storage plugin: %v", err)
	}

	// Register plugin
	if err := app.RegisterPlugin(storage, "storage"); err != nil {
		log.Fatalf("Failed to register plugin: %v", err)
	}
	fmt.Printf("Storage plugin registered with buckets: %v\n", storage.Buckets())

	// Create modules
	documentModule := NewDocumentModule()
	uploadModule := NewUploadModule()
	mediaModule := NewMediaModule()

	// Register modules
	app.Register(documentModule)
	app.Register(uploadModule)
	app.Register(mediaModule)

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Println()

	// Demo: Document Storage
	fmt.Println("=== Document Storage Demo ===")

	// Store documents
	doc1 := []byte("This is the Q4 2024 financial report content...")
	info, _ := documentModule.StoreDocument(ctx, "reports/2024/q4-report.txt", doc1, "text/plain", "finance-team")
	fmt.Printf("Stored document: %s (%d bytes)\n", info.Name, info.Size)

	doc2 := []byte("Employee handbook content goes here...")
	info, _ = documentModule.StoreDocument(ctx, "hr/handbook.txt", doc2, "text/plain", "hr-team")
	fmt.Printf("Stored document: %s (%d bytes)\n", info.Name, info.Size)

	// List all documents
	docs, _ := documentModule.ListDocuments()
	fmt.Printf("Total documents: %d\n", len(docs))
	for _, doc := range docs {
		fmt.Printf("  - %s (%d bytes, hash: %s)\n", doc.Name, doc.Size, doc.Digest[:16]+"...")
	}

	// List by prefix
	reports, _ := documentModule.ListDocumentsByPrefix("reports/")
	fmt.Printf("Reports: %d\n", len(reports))

	// Retrieve document
	content, _ := documentModule.GetDocument("reports/2024/q4-report.txt")
	fmt.Printf("Retrieved content: %s...\n", string(content)[:30])
	fmt.Println()

	// Demo: File Uploads
	fmt.Println("=== File Upload Demo ===")

	// Small file upload
	smallFile := []byte("Small file content for upload")
	uploadInfo, _ := uploadModule.UploadFile(ctx, "notes.txt", smallFile)
	fmt.Printf("Uploaded: %s (%d bytes)\n", uploadInfo.Name, uploadInfo.Size)

	// Large file upload using streaming
	largeContent := bytes.Repeat([]byte("Large content block. "), 1000)
	reader := bytes.NewReader(largeContent)
	uploadInfo, _ = uploadModule.UploadLargeFile("large-data.bin", reader)
	fmt.Printf("Uploaded large file: %s (%d bytes)\n", uploadInfo.Name, uploadInfo.Size)

	// List uploads
	uploads, _ := uploadModule.ListUploads()
	fmt.Printf("Total uploads: %d\n", len(uploads))

	// Stream download
	streamReader, streamInfo, _ := uploadModule.StreamFile(uploadInfo.Name)
	defer streamReader.Close()
	fmt.Printf("Streaming file: %s (%d bytes)\n", streamInfo.Name, streamInfo.Size)

	// Read first 50 bytes
	buf := make([]byte, 50)
	n, _ := streamReader.Read(buf)
	fmt.Printf("First %d bytes: %s...\n", n, string(buf[:n]))
	fmt.Println()

	// Demo: Media with Multiple Buckets
	fmt.Println("=== Media Module Demo (Multiple Buckets) ===")

	// Store image with thumbnail
	imageData := []byte("Fake image data representing a JPEG file...")
	thumbData := []byte("Fake thumbnail data...")
	mediaModule.StoreImage(ctx, "photo001.jpg", imageData, thumbData)

	// Retrieve
	img, _ := mediaModule.GetImage("photo001.jpg")
	thumb, _ := mediaModule.GetThumbnail("photo001.jpg")
	fmt.Printf("Image size: %d bytes, Thumbnail size: %d bytes\n", len(img), len(thumb))

	// Wait for shutdown signal
	fmt.Println("\nPress Ctrl+C to shutdown...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}

	fmt.Println("App stopped successfully")
}
