package fileops

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements types.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any)          {}
func (m *mockLogger) Info(msg string, args ...any)           {}
func (m *mockLogger) Warn(msg string, args ...any)           {}
func (m *mockLogger) Error(msg string, args ...any)          {}
func (m *mockLogger) With(args ...any) types.Logger          { return m }
func (m *mockLogger) WithError(err error) types.Logger       { return m }
func (m *mockLogger) WithModule(module string) types.Logger { return m }

// createTestModule creates a module with an in-memory bucket for testing
func createTestModule(t *testing.T) (*Module, fsjetstream.FileStoragePort) {
	// For now, we'll create a simple module without full plugin initialization
	// In a full integration test, you would initialize the plugin with a NATS connection

	// Create mono application with embedded NATS for testing
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError), // Suppress logs in tests
	)
	require.NoError(t, err)

	// Create fs-jetstream plugin with in-memory storage
	plugin, err := fsjetstream.New(fsjetstream.Config{
		Buckets: []fsjetstream.BucketConfig{
			{
				Name:        "user-settings",
				Description: "Test bucket",
				MaxBytes:    10 * 1024 * 1024, // 10MB
				Storage:     fsjetstream.MemoryStorage,
			},
		},
	})
	require.NoError(t, err)

	// Register plugin with the app
	err = app.RegisterPlugin(plugin, "storage")
	require.NoError(t, err)

	// Start the application (this initializes NATS and plugins)
	err = app.Start(context.Background())
	require.NoError(t, err)

	// Cleanup after test
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	// Create module and inject plugin
	module := NewModule(&mockLogger{})
	module.SetPlugin("storage", plugin)

	// Start module
	err = module.Start(context.Background())
	require.NoError(t, err)

	return module, module.bucket
}

// TestHandleFileSave_Success tests successful file save operation
func TestHandleFileSave_Success(t *testing.T) {
	module, _ := createTestModule(t)

	req := FileSaveRequest{
		Filename: "test-settings.json",
		Content: map[string]any{
			"theme": "dark",
			"lang":  "en",
		},
	}

	// Call handler directly with typed signature
	resp, err := module.handleFileSave(context.Background(), req, nil)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.FileID)
	assert.Equal(t, "test-settings.json", resp.Filename)
	assert.Greater(t, resp.Size, int64(0))
	assert.NotEmpty(t, resp.Digest)
	assert.Empty(t, resp.Error)
}

// TestHandleFileSave_EmptyFilename tests error handling for empty filename
func TestHandleFileSave_EmptyFilename(t *testing.T) {
	module, _ := createTestModule(t)

	req := FileSaveRequest{
		Filename: "",
		Content: map[string]any{
			"theme": "dark",
		},
	}

	resp, err := module.handleFileSave(context.Background(), req, nil)
	require.NoError(t, err)

	assert.Contains(t, resp.Error, "filename is required")
}

// TestHandleFileSave_NullContent tests error handling for null content
func TestHandleFileSave_NullContent(t *testing.T) {
	module, _ := createTestModule(t)

	req := FileSaveRequest{
		Filename: "test.json",
		Content:  nil,
	}

	resp, err := module.handleFileSave(context.Background(), req, nil)
	require.NoError(t, err)

	assert.Contains(t, resp.Error, "content is required")
}

// TestHandleFileArchive_Success tests successful file archive operation
func TestHandleFileArchive_Success(t *testing.T) {
	module, bucket := createTestModule(t)

	// First, save a file to archive
	content := map[string]any{"theme": "dark"}
	contentBytes, err := json.Marshal(content)
	require.NoError(t, err)

	fileID := "test-file-id"
	storageKey := fileID + "/settings.json"
	_, err = bucket.Put(context.Background(), storageKey, contentBytes)
	require.NoError(t, err)

	// Now archive it
	archiveReq := FileArchiveRequest{
		FileID: fileID,
	}

	reqBytes, err := json.Marshal(archiveReq)
	require.NoError(t, err)

	msg := &mono.Msg{
		Data: reqBytes,
	}

	archiveErr := module.handleFileArchive(context.Background(), msg)
	assert.NoError(t, archiveErr)

	// Verify ZIP file exists
	files, err := bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	require.NoError(t, err)

	// Should have one file: the ZIP
	assert.Len(t, files, 1)
	assert.Contains(t, files[0].Name, ".zip")

	// Verify original JSON file was deleted
	assert.NotContains(t, files[0].Name, ".json")
}

// TestHandleFileArchive_FileNotFound tests handling of non-existent file
func TestHandleFileArchive_FileNotFound(t *testing.T) {
	module, _ := createTestModule(t)

	archiveReq := FileArchiveRequest{
		FileID: "non-existent-file",
	}

	reqBytes, err := json.Marshal(archiveReq)
	require.NoError(t, err)

	msg := &mono.Msg{
		Data: reqBytes,
	}

	// Should not return error (fire-and-forget pattern)
	notFoundErr := module.handleFileArchive(context.Background(), msg)
	assert.NoError(t, notFoundErr)
}

// TestHandleFileArchive_InvalidJSON tests handling of malformed request
func TestHandleFileArchive_InvalidJSON(t *testing.T) {
	module, _ := createTestModule(t)

	msg := &mono.Msg{
		Data: []byte("invalid json"),
	}

	// Should not return error (fire-and-forget pattern)
	invalidErr := module.handleFileArchive(context.Background(), msg)
	assert.NoError(t, invalidErr)
}

// TestHandleFileArchive_AlreadyArchived tests skipping already archived files
func TestHandleFileArchive_AlreadyArchived(t *testing.T) {
	module, bucket := createTestModule(t)

	// Save a ZIP file directly
	fileID := "test-zip-id"
	storageKey := fileID + "/settings.zip"
	_, err := bucket.Put(context.Background(), storageKey, []byte("fake zip content"))
	require.NoError(t, err)

	// Try to archive it
	archiveReq := FileArchiveRequest{
		FileID: fileID,
	}

	reqBytes, err := json.Marshal(archiveReq)
	require.NoError(t, err)

	msg := &mono.Msg{
		Data: reqBytes,
	}

	archiveErr := module.handleFileArchive(context.Background(), msg)
	assert.NoError(t, archiveErr)

	// Should still have only one file (not duplicated)
	files, err := bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}
