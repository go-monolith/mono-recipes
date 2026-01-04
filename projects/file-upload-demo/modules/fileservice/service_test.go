package fileservice

import (
	"testing"
	"time"
)

// These tests focus on the data types and validation logic.
// Integration tests with the actual fs-jetstream plugin are recommended
// for full end-to-end testing.

func TestFileInfoFields(t *testing.T) {
	info := FileInfo{
		ID:          "test-id",
		Name:        "test.pdf",
		Size:        1024,
		ContentType: "application/pdf",
		Digest:      "sha256:abc123",
		CreatedAt:   time.Now(),
		Headers: map[string]string{
			"Custom-Header": "value",
		},
	}

	if info.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", info.ID)
	}

	if info.Name != "test.pdf" {
		t.Errorf("Expected Name 'test.pdf', got '%s'", info.Name)
	}

	if info.Size != 1024 {
		t.Errorf("Expected Size 1024, got %d", info.Size)
	}

	if info.ContentType != "application/pdf" {
		t.Errorf("Expected ContentType 'application/pdf', got '%s'", info.ContentType)
	}

	if info.Digest != "sha256:abc123" {
		t.Errorf("Expected Digest 'sha256:abc123', got '%s'", info.Digest)
	}

	if info.Headers["Custom-Header"] != "value" {
		t.Errorf("Expected Custom-Header 'value', got '%s'", info.Headers["Custom-Header"])
	}
}

func TestUploadResultFields(t *testing.T) {
	result := UploadResult{
		FileInfo: FileInfo{
			ID:          "test-id",
			Name:        "test.txt",
			Size:        100,
			ContentType: "text/plain",
		},
		Message:    "File uploaded successfully",
		DurationMs: 150,
	}

	if result.Message != "File uploaded successfully" {
		t.Errorf("Expected Message 'File uploaded successfully', got '%s'", result.Message)
	}

	if result.DurationMs != 150 {
		t.Errorf("Expected DurationMs 150, got %d", result.DurationMs)
	}

	if result.FileInfo.ID != "test-id" {
		t.Errorf("Expected FileInfo.ID 'test-id', got '%s'", result.FileInfo.ID)
	}

	if result.FileInfo.Name != "test.txt" {
		t.Errorf("Expected FileInfo.Name 'test.txt', got '%s'", result.FileInfo.Name)
	}
}

func TestListResultFields(t *testing.T) {
	result := ListResult{
		Files: []FileInfo{
			{ID: "id1", Name: "file1.txt", Size: 100, ContentType: "text/plain"},
			{ID: "id2", Name: "file2.txt", Size: 200, ContentType: "text/plain"},
			{ID: "id3", Name: "file3.pdf", Size: 1024, ContentType: "application/pdf"},
		},
		Total:      3,
		DurationMs: 50,
	}

	if result.Total != 3 {
		t.Errorf("Expected Total 3, got %d", result.Total)
	}

	if len(result.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(result.Files))
	}

	if result.DurationMs != 50 {
		t.Errorf("Expected DurationMs 50, got %d", result.DurationMs)
	}

	// Verify file entries
	if result.Files[0].ID != "id1" {
		t.Errorf("Expected first file ID 'id1', got '%s'", result.Files[0].ID)
	}

	if result.Files[1].Name != "file2.txt" {
		t.Errorf("Expected second file name 'file2.txt', got '%s'", result.Files[1].Name)
	}

	if result.Files[2].ContentType != "application/pdf" {
		t.Errorf("Expected third file content type 'application/pdf', got '%s'", result.Files[2].ContentType)
	}
}

func TestDownloadResultFields(t *testing.T) {
	result := DownloadResult{
		FileInfo: FileInfo{
			ID:          "download-id",
			Name:        "document.pdf",
			Size:        5000,
			ContentType: "application/pdf",
			Digest:      "sha256:xyz789",
		},
		DurationMs: 25,
	}

	if result.FileInfo.ID != "download-id" {
		t.Errorf("Expected FileInfo.ID 'download-id', got '%s'", result.FileInfo.ID)
	}

	if result.FileInfo.Size != 5000 {
		t.Errorf("Expected FileInfo.Size 5000, got %d", result.FileInfo.Size)
	}

	if result.DurationMs != 25 {
		t.Errorf("Expected DurationMs 25, got %d", result.DurationMs)
	}
}

func TestFileInfoWithEmptyHeaders(t *testing.T) {
	info := FileInfo{
		ID:          "empty-headers",
		Name:        "file.txt",
		Size:        50,
		ContentType: "text/plain",
		Headers:     nil,
	}

	if info.Headers != nil {
		t.Error("Expected Headers to be nil")
	}

	// Test with empty map
	info.Headers = make(map[string]string)
	if len(info.Headers) != 0 {
		t.Error("Expected Headers to be empty map")
	}
}

func TestFileInfoCreatedAtTimestamp(t *testing.T) {
	now := time.Now()
	info := FileInfo{
		ID:        "time-test",
		Name:      "file.txt",
		CreatedAt: now,
	}

	if !info.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt to equal %v, got %v", now, info.CreatedAt)
	}

	// Test that zero time works
	zeroInfo := FileInfo{
		ID:   "zero-time",
		Name: "file.txt",
	}

	if !zeroInfo.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be zero value")
	}
}

func TestListResultEmpty(t *testing.T) {
	result := ListResult{
		Files:      []FileInfo{},
		Total:      0,
		DurationMs: 5,
	}

	if result.Total != 0 {
		t.Errorf("Expected Total 0, got %d", result.Total)
	}

	if len(result.Files) != 0 {
		t.Errorf("Expected empty files slice, got %d files", len(result.Files))
	}
}

func TestUploadResultZeroDuration(t *testing.T) {
	result := UploadResult{
		FileInfo: FileInfo{
			ID:   "fast-upload",
			Name: "tiny.txt",
			Size: 10,
		},
		Message:    "Done",
		DurationMs: 0,
	}

	if result.DurationMs != 0 {
		t.Errorf("Expected DurationMs 0, got %d", result.DurationMs)
	}
}

func TestFileInfoLargeSize(t *testing.T) {
	// Test with a large file size (5GB)
	largeSize := int64(5 * 1024 * 1024 * 1024)

	info := FileInfo{
		ID:   "large-file",
		Name: "bigfile.bin",
		Size: largeSize,
	}

	if info.Size != largeSize {
		t.Errorf("Expected Size %d, got %d", largeSize, info.Size)
	}
}

func TestFileInfoMultipleHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type":     "image/png",
		"Content-Encoding": "gzip",
		"Cache-Control":    "max-age=3600",
		"X-Custom-Header":  "custom-value",
	}

	info := FileInfo{
		ID:          "multi-header",
		Name:        "image.png",
		Size:        1024,
		ContentType: "image/png",
		Headers:     headers,
	}

	if len(info.Headers) != 4 {
		t.Errorf("Expected 4 headers, got %d", len(info.Headers))
	}

	if info.Headers["Content-Encoding"] != "gzip" {
		t.Errorf("Expected Content-Encoding 'gzip', got '%s'", info.Headers["Content-Encoding"])
	}

	if info.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", info.Headers["X-Custom-Header"])
	}
}
