package fileops

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"node-nats-client-demo/modules/fileops/contract"
)

// The fileops module needs the fs-jetstream plugin, which monotest does not
// emulate, so these tests run it in a real application: embedded NATS on a free
// port, JetStream in a temp dir and strict module boundaries.

// mockLogger implements types.Logger for testing.
type mockLogger struct{}

var _ types.Logger = (*mockLogger)(nil)

func (m *mockLogger) Debug(msg string, args ...any)         {}
func (m *mockLogger) Info(msg string, args ...any)          {}
func (m *mockLogger) Warn(msg string, args ...any)          {}
func (m *mockLogger) Error(msg string, args ...any)         {}
func (m *mockLogger) With(args ...any) types.Logger         { return m }
func (m *mockLogger) WithError(err error) types.Logger      { return m }
func (m *mockLogger) WithModule(module string) types.Logger { return m }

// freePort returns a TCP port that is free right now, so the test does not
// collide with a running demo binary on the default 4222.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

// startTestApp starts an application with the storage plugin and the fileops
// module registered, as main.go does, and returns both.
func startTestApp(t *testing.T) (mono.MonoApplication, *Module) {
	t.Helper()

	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError), // Suppress logs in tests
		mono.WithNATSPort(freePort(t)),
		mono.WithJetStreamStorageDir(t.TempDir()),
		// Default (non-strict) mode on purpose: violations are recorded in
		// app.ModuleBoundaries() instead of rejected, so the report check below
		// is meaningful. main.go runs strict once this report is clean.
	)
	require.NoError(t, err)

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
	require.NoError(t, app.RegisterPlugin(plugin, "storage"))

	module := NewModule(&mockLogger{})
	require.NoError(t, app.Register(module))

	require.NoError(t, app.Start(context.Background()))
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	return app, module
}

// requireNoBoundaryViolations checks the application's boundary report.
func requireNoBoundaryViolations(t *testing.T, app mono.MonoApplication) {
	t.Helper()
	for _, r := range app.ModuleBoundaries() {
		assert.Empty(t, r.Violations, "module %s has boundary violations", r.Module)
	}
}

// waitForZip polls the bucket until fileID has been archived as a ZIP.
func waitForZip(t *testing.T, bucket fsjetstream.FileStoragePort, fileID string) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		files, err := bucket.List(fsjetstream.WithPrefix(fileID + "/"))
		require.NoError(t, err)
		names := make([]string, 0, len(files))
		zipped := false
		for _, f := range files {
			names = append(names, f.Name)
			zipped = zipped || strings.HasSuffix(f.Name, ".zip")
		}
		if zipped {
			return names
		}
		if time.Now().After(deadline) {
			t.Fatalf("file %s not archived in time, bucket has %v", fileID, names)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestFileOpsModuleInApp drives both services through their contracts, the way
// the Node.js clients reach them (same subjects, same JSON), and checks the
// module boundary report.
func TestFileOpsModuleInApp(t *testing.T) {
	app, module := startTestApp(t)
	ctx := context.Background()
	services := app.Services("fileops")

	saved, err := contract.Save.Call(ctx, services, contract.FileSaveRequest{
		Filename: "settings.json",
		Content:  map[string]any{"theme": "dark"},
	})
	require.NoError(t, err)
	require.Empty(t, saved.Error)
	require.NotEmpty(t, saved.FileID)

	require.NoError(t, contract.Archive.Send(ctx, services, contract.FileArchiveRequest{FileID: saved.FileID}))
	names := waitForZip(t, module.bucket, saved.FileID)
	assert.Equal(t, []string{saved.FileID + "/settings.zip"}, names)

	requireNoBoundaryViolations(t, app)
}

// TestArchive_InvalidJSONIsDropped sends a payload that does not decode to the
// archive service: the contract drops it, and the worker keeps serving.
func TestArchive_InvalidJSONIsDropped(t *testing.T) {
	app, module := startTestApp(t)
	ctx := context.Background()
	services := app.Services("fileops")

	client, err := services.GetQueueGroupService(contract.Archive.Name)
	require.NoError(t, err)
	require.NoError(t, client.Send(ctx, []byte("invalid json")))

	fileID := "after-invalid"
	_, err = module.bucket.Put(ctx, fileID+"/settings.json", []byte(`{"theme":"dark"}`))
	require.NoError(t, err)
	require.NoError(t, contract.Archive.Send(ctx, services, contract.FileArchiveRequest{FileID: fileID}))
	waitForZip(t, module.bucket, fileID)

	requireNoBoundaryViolations(t, app)
}

// TestHandleFileSave_Success tests successful file save operation
func TestHandleFileSave_Success(t *testing.T) {
	_, module := startTestApp(t)

	req := contract.FileSaveRequest{
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
	_, module := startTestApp(t)

	req := contract.FileSaveRequest{
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
	_, module := startTestApp(t)

	req := contract.FileSaveRequest{
		Filename: "test.json",
		Content:  nil,
	}

	resp, err := module.handleFileSave(context.Background(), req, nil)
	require.NoError(t, err)

	assert.Contains(t, resp.Error, "content is required")
}

// TestHandleFileArchive_Success tests successful file archive operation
func TestHandleFileArchive_Success(t *testing.T) {
	_, module := startTestApp(t)
	bucket := module.bucket

	// First, save a file to archive
	content := map[string]any{"theme": "dark"}
	contentBytes, err := json.Marshal(content)
	require.NoError(t, err)

	fileID := "test-file-id"
	storageKey := fileID + "/settings.json"
	_, err = bucket.Put(context.Background(), storageKey, contentBytes)
	require.NoError(t, err)

	// Now archive it
	archiveErr := module.handleFileArchive(context.Background(), contract.FileArchiveRequest{FileID: fileID}, nil)
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
	_, module := startTestApp(t)

	// Should not return error (fire-and-forget pattern)
	notFoundErr := module.handleFileArchive(context.Background(), contract.FileArchiveRequest{FileID: "non-existent-file"}, nil)
	assert.NoError(t, notFoundErr)
}

// TestHandleFileArchive_MissingFileID tests handling of an empty file ID
func TestHandleFileArchive_MissingFileID(t *testing.T) {
	_, module := startTestApp(t)

	// Should not return error (fire-and-forget pattern)
	assert.NoError(t, module.handleFileArchive(context.Background(), contract.FileArchiveRequest{}, nil))
}

// TestHandleFileArchive_AlreadyArchived tests skipping already archived files
func TestHandleFileArchive_AlreadyArchived(t *testing.T) {
	_, module := startTestApp(t)
	bucket := module.bucket

	// Save a ZIP file directly
	fileID := "test-zip-id"
	storageKey := fileID + "/settings.zip"
	_, err := bucket.Put(context.Background(), storageKey, []byte("fake zip content"))
	require.NoError(t, err)

	// Try to archive it
	archiveErr := module.handleFileArchive(context.Background(), contract.FileArchiveRequest{FileID: fileID}, nil)
	assert.NoError(t, archiveErr)

	// Should still have only one file (not duplicated)
	files, err := bucket.List(fsjetstream.WithPrefix(fileID + "/"))
	require.NoError(t, err)
	assert.Len(t, files, 1)
}
