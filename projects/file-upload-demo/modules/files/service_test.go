package files

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockObjectStore is a mock implementation of ObjectStore for testing.
type mockObjectStore struct {
	objects map[string]mockObject
}

type mockObject struct {
	data        []byte
	contentType string
	modTime     time.Time
}

func newMockObjectStore() *mockObjectStore {
	return &mockObjectStore{
		objects: make(map[string]mockObject),
	}
}

func (m *mockObjectStore) Put(_ context.Context, name string, data []byte, contentType string) (*ObjectInfo, error) {
	m.objects[name] = mockObject{
		data:        data,
		contentType: contentType,
		modTime:     time.Now(),
	}
	return &ObjectInfo{
		Name:        name,
		Size:        uint64(len(data)),
		ContentType: contentType,
		ModTime:     m.objects[name].modTime,
	}, nil
}

func (m *mockObjectStore) Get(_ context.Context, name string) ([]byte, *ObjectInfo, error) {
	obj, ok := m.objects[name]
	if !ok {
		return nil, nil, fmt.Errorf("object not found: %s", name)
	}
	return obj.data, &ObjectInfo{
		Name:        name,
		Size:        uint64(len(obj.data)),
		ContentType: obj.contentType,
		ModTime:     obj.modTime,
	}, nil
}

func (m *mockObjectStore) Delete(_ context.Context, name string) error {
	if _, ok := m.objects[name]; !ok {
		return fmt.Errorf("object not found: %s", name)
	}
	delete(m.objects, name)
	return nil
}

func (m *mockObjectStore) List(_ context.Context) ([]*ObjectInfo, error) {
	objects := make([]*ObjectInfo, 0, len(m.objects))
	for name, obj := range m.objects {
		objects = append(objects, &ObjectInfo{
			Name:        name,
			Size:        uint64(len(obj.data)),
			ContentType: obj.contentType,
			ModTime:     obj.modTime,
		})
	}
	return objects, nil
}

func (m *mockObjectStore) GetInfo(_ context.Context, name string) (*ObjectInfo, error) {
	obj, ok := m.objects[name]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", name)
	}
	return &ObjectInfo{
		Name:        name,
		Size:        uint64(len(obj.data)),
		ContentType: obj.contentType,
		ModTime:     obj.modTime,
	}, nil
}

func TestService_Upload(t *testing.T) {
	store := newMockObjectStore()
	service := NewService(store)
	ctx := context.Background()

	tests := []struct {
		name        string
		fileName    string
		data        []byte
		contentType string
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful upload",
			fileName:    "test.txt",
			data:        []byte("hello world"),
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "empty file name",
			fileName:    "",
			data:        []byte("hello world"),
			contentType: "text/plain",
			wantErr:     true,
			errContains: "file name is required",
		},
		{
			name:        "empty data",
			fileName:    "empty.txt",
			data:        []byte{},
			contentType: "text/plain",
			wantErr:     true,
			errContains: "file data is empty",
		},
		{
			name:        "default content type",
			fileName:    "binary.bin",
			data:        []byte{0x00, 0x01, 0x02},
			contentType: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := service.Upload(ctx, tt.fileName, tt.data, tt.contentType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Upload() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Upload() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("Upload() unexpected error: %v", err)
				return
			}
			if meta.ID == "" {
				t.Error("Upload() meta.ID is empty")
			}
			if meta.Name != tt.fileName {
				t.Errorf("Upload() meta.Name = %q, want %q", meta.Name, tt.fileName)
			}
			if meta.Size != int64(len(tt.data)) {
				t.Errorf("Upload() meta.Size = %d, want %d", meta.Size, len(tt.data))
			}
			expectedContentType := tt.contentType
			if expectedContentType == "" {
				expectedContentType = "application/octet-stream"
			}
			if meta.ContentType != expectedContentType {
				t.Errorf("Upload() meta.ContentType = %q, want %q", meta.ContentType, expectedContentType)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	store := newMockObjectStore()
	service := NewService(store)
	ctx := context.Background()

	// Upload a file first
	testData := []byte("test content")
	meta, err := service.Upload(ctx, "test.txt", testData, "text/plain")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name        string
		fileID      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful get",
			fileID:  meta.ID,
			wantErr: false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			wantErr:     true,
			errContains: "file ID is required",
		},
		{
			name:        "non-existent file",
			fileID:      "non-existent-id",
			wantErr:     true,
			errContains: "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, fileMeta, err := service.Get(ctx, tt.fileID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Get() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Get() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("Get() unexpected error: %v", err)
				return
			}
			if string(data) != string(testData) {
				t.Errorf("Get() data = %q, want %q", string(data), string(testData))
			}
			if fileMeta.ID != meta.ID {
				t.Errorf("Get() fileMeta.ID = %q, want %q", fileMeta.ID, meta.ID)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	store := newMockObjectStore()
	service := NewService(store)
	ctx := context.Background()

	// Upload some files
	files := []struct {
		name        string
		data        []byte
		contentType string
	}{
		{"file1.txt", []byte("content 1"), "text/plain"},
		{"file2.txt", []byte("content 2"), "text/plain"},
		{"file3.json", []byte(`{"key":"value"}`), "application/json"},
	}

	for _, f := range files {
		_, err := service.Upload(ctx, f.name, f.data, f.contentType)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		wantCount int
		wantTotal int
	}{
		{
			name:      "list all",
			limit:     0,
			offset:    0,
			wantCount: 3,
			wantTotal: 3,
		},
		{
			name:      "limit 2",
			limit:     2,
			offset:    0,
			wantCount: 2,
			wantTotal: 3,
		},
		{
			name:      "offset 1",
			limit:     0,
			offset:    1,
			wantCount: 2,
			wantTotal: 3,
		},
		{
			name:      "limit and offset",
			limit:     1,
			offset:    1,
			wantCount: 1,
			wantTotal: 3,
		},
		{
			name:      "offset beyond total",
			limit:     0,
			offset:    10,
			wantCount: 0,
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, total, err := service.List(ctx, tt.limit, tt.offset)
			if err != nil {
				t.Errorf("List() unexpected error: %v", err)
				return
			}
			if len(result) != tt.wantCount {
				t.Errorf("List() returned %d files, want %d", len(result), tt.wantCount)
			}
			if total != tt.wantTotal {
				t.Errorf("List() total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	store := newMockObjectStore()
	service := NewService(store)
	ctx := context.Background()

	// Upload a file first
	meta, err := service.Upload(ctx, "to-delete.txt", []byte("delete me"), "text/plain")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name        string
		fileID      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful delete",
			fileID:  meta.ID,
			wantErr: false,
		},
		{
			name:        "empty file ID",
			fileID:      "",
			wantErr:     true,
			errContains: "file ID is required",
		},
		{
			name:        "non-existent file",
			fileID:      "non-existent-id",
			wantErr:     true,
			errContains: "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Delete(ctx, tt.fileID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Delete() expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Delete() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("Delete() unexpected error: %v", err)
				return
			}

			// Verify file is deleted
			_, _, err = service.Get(ctx, tt.fileID)
			if err == nil {
				t.Error("Delete() file still exists after deletion")
			}
		})
	}
}

func TestParseStorageName(t *testing.T) {
	tests := []struct {
		input    string
		wantID   string
		wantName string
	}{
		{
			input:    "abc123/file.txt",
			wantID:   "abc123",
			wantName: "file.txt",
		},
		{
			input:    "uuid-1234-5678/document.pdf",
			wantID:   "uuid-1234-5678",
			wantName: "document.pdf",
		},
		{
			input:    "no-separator",
			wantID:   "",
			wantName: "no-separator",
		},
		{
			input:    "id/path/with/slashes.txt",
			wantID:   "id",
			wantName: "path/with/slashes.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotID, gotName := parseStorageName(tt.input)
			if gotID != tt.wantID {
				t.Errorf("parseStorageName() id = %q, want %q", gotID, tt.wantID)
			}
			if gotName != tt.wantName {
				t.Errorf("parseStorageName() name = %q, want %q", gotName, tt.wantName)
			}
		})
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
