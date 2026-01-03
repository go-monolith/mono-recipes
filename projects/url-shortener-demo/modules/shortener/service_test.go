package shortener

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/example/url-shortener-demo/domain/url"
)

// mockKVStore implements KVStore for testing.
type mockKVStore struct {
	data   map[string]*domain.ShortURL
	putErr error
	getErr error
}

func newMockKVStore() *mockKVStore {
	return &mockKVStore{
		data: make(map[string]*domain.ShortURL),
	}
}

func (m *mockKVStore) Put(_ context.Context, shortCode string, url *domain.ShortURL) error {
	if m.putErr != nil {
		return m.putErr
	}
	m.data[shortCode] = url
	return nil
}

func (m *mockKVStore) Get(_ context.Context, shortCode string) (*domain.ShortURL, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	url, ok := m.data[shortCode]
	if !ok {
		return nil, errors.New("short code not found")
	}
	return url, nil
}

func (m *mockKVStore) Delete(_ context.Context, shortCode string) error {
	delete(m.data, shortCode)
	return nil
}

func (m *mockKVStore) Exists(_ context.Context, shortCode string) (bool, error) {
	_, ok := m.data[shortCode]
	return ok, nil
}

// mockStatsStore implements StatsStore for testing.
type mockStatsStore struct {
	data   map[string]*domain.URLStats
	setErr error
	getErr error
}

func newMockStatsStore() *mockStatsStore {
	return &mockStatsStore{
		data: make(map[string]*domain.URLStats),
	}
}

func (m *mockStatsStore) IncrementAccess(_ context.Context, shortCode string) error {
	stats, ok := m.data[shortCode]
	if !ok {
		stats = &domain.URLStats{
			ShortCode:   shortCode,
			AccessCount: 0,
			CreatedAt:   time.Now(),
		}
	}
	stats.AccessCount++
	now := time.Now()
	stats.LastAccess = &now
	m.data[shortCode] = stats
	return nil
}

func (m *mockStatsStore) GetStats(_ context.Context, shortCode string) (*domain.URLStats, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	stats, ok := m.data[shortCode]
	if !ok {
		return nil, errors.New("stats not found")
	}
	return stats, nil
}

func (m *mockStatsStore) SetStats(_ context.Context, shortCode string, stats *domain.URLStats) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.data[shortCode] = stats
	return nil
}

func TestService_Shorten(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	tests := []struct {
		name        string
		url         string
		customCode  string
		ttlSeconds  int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful shorten with auto-generated code",
			url:         "https://example.com/path",
			customCode:  "",
			ttlSeconds:  0,
			expectError: false,
		},
		{
			name:        "successful shorten with custom code",
			url:         "https://example.com/path2",
			customCode:  "mycode",
			ttlSeconds:  0,
			expectError: false,
		},
		{
			name:        "successful shorten with TTL",
			url:         "https://example.com/path3",
			customCode:  "withttl",
			ttlSeconds:  3600,
			expectError: false,
		},
		{
			name:        "empty URL",
			url:         "",
			customCode:  "",
			ttlSeconds:  0,
			expectError: true,
			errorMsg:    "URL is required",
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			customCode:  "",
			ttlSeconds:  0,
			expectError: true,
			errorMsg:    "invalid URL format",
		},
		{
			name:        "invalid custom code format",
			url:         "https://example.com",
			customCode:  "invalid_code",
			ttlSeconds:  0,
			expectError: true,
			errorMsg:    "invalid custom code",
		},
		{
			name:        "custom code too long",
			url:         "https://example.com",
			customCode:  "123456789012345678901",
			ttlSeconds:  0,
			expectError: true,
			errorMsg:    "invalid custom code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Shorten(ctx, tt.url, tt.customCode, tt.ttlSeconds)

			if tt.expectError {
				if err == nil {
					t.Errorf("Shorten() expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("Shorten() error = %q, want error containing %q", err.Error(), tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Shorten() unexpected error = %v", err)
			}

			if result == nil {
				t.Fatal("Shorten() returned nil result")
			}

			// Verify result fields
			if result.OriginalURL != tt.url {
				t.Errorf("Shorten() OriginalURL = %q, want %q", result.OriginalURL, tt.url)
			}

			if tt.customCode != "" && result.ShortCode != tt.customCode {
				t.Errorf("Shorten() ShortCode = %q, want %q", result.ShortCode, tt.customCode)
			}

			if result.ID == "" {
				t.Error("Shorten() ID should not be empty")
			}

			if result.CreatedAt.IsZero() {
				t.Error("Shorten() CreatedAt should not be zero")
			}

			if tt.ttlSeconds > 0 {
				if result.ExpiresAt == nil {
					t.Error("Shorten() ExpiresAt should be set when TTL is provided")
				}
			}
		})
	}
}

func TestService_Shorten_DuplicateCustomCode(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	// First shorten with custom code
	_, err := service.Shorten(ctx, "https://example.com/first", "mycode", 0)
	if err != nil {
		t.Fatalf("First Shorten() error = %v", err)
	}

	// Try to use the same custom code
	_, err = service.Shorten(ctx, "https://example.com/second", "mycode", 0)
	if err == nil {
		t.Error("Shorten() expected error for duplicate custom code")
	}
	if !containsSubstring(err.Error(), "already in use") {
		t.Errorf("Shorten() error = %q, want error containing 'already in use'", err.Error())
	}
}

func TestService_Resolve(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	// Create a short URL first
	result, err := service.Shorten(ctx, "https://example.com/original", "resolvetest", 0)
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}

	tests := []struct {
		name        string
		shortCode   string
		expectError bool
		expectedURL string
	}{
		{
			name:        "successful resolve",
			shortCode:   result.ShortCode,
			expectError: false,
			expectedURL: "https://example.com/original",
		},
		{
			name:        "resolve non-existent code",
			shortCode:   "nonexist",
			expectError: true,
		},
		{
			name:        "resolve invalid code format",
			shortCode:   "invalid_code",
			expectError: true,
		},
		{
			name:        "resolve empty code",
			shortCode:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := service.Resolve(ctx, tt.shortCode)

			if tt.expectError {
				if err == nil {
					t.Error("Resolve() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Resolve() unexpected error = %v", err)
			}

			if url != tt.expectedURL {
				t.Errorf("Resolve() = %q, want %q", url, tt.expectedURL)
			}
		})
	}
}

func TestService_RecordAccess(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	shortCode := "accesstest"

	// Record access multiple times
	for i := 0; i < 5; i++ {
		err := service.RecordAccess(ctx, shortCode)
		if err != nil {
			t.Fatalf("RecordAccess() error = %v", err)
		}
	}

	// Verify access count
	stats, err := statsStore.GetStats(ctx, shortCode)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.AccessCount != 5 {
		t.Errorf("AccessCount = %d, want 5", stats.AccessCount)
	}

	if stats.LastAccess == nil {
		t.Error("LastAccess should be set")
	}
}

func TestService_GetStats(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	// Create a short URL
	result, err := service.Shorten(ctx, "https://example.com/stats", "statstest", 0)
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}

	// Record some accesses
	for i := 0; i < 3; i++ {
		_ = service.RecordAccess(ctx, result.ShortCode)
	}

	// Get stats
	stats, err := service.GetStats(ctx, result.ShortCode)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.ShortCode != result.ShortCode {
		t.Errorf("ShortCode = %q, want %q", stats.ShortCode, result.ShortCode)
	}

	if stats.OriginalURL != "https://example.com/stats" {
		t.Errorf("OriginalURL = %q, want %q", stats.OriginalURL, "https://example.com/stats")
	}

	if stats.AccessCount != 3 {
		t.Errorf("AccessCount = %d, want 3", stats.AccessCount)
	}
}

func TestService_GetStats_NotFound(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	_, err := service.GetStats(ctx, "nonexistent")
	if err == nil {
		t.Error("GetStats() expected error for non-existent code")
	}
}

func TestService_GetStats_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	_, err := service.GetStats(ctx, "invalid_format")
	if err == nil {
		t.Error("GetStats() expected error for invalid code format")
	}
}

func TestService_GetFullShortURL(t *testing.T) {
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()

	tests := []struct {
		name      string
		baseURL   string
		shortCode string
		expected  string
	}{
		{
			name:      "standard base URL",
			baseURL:   "http://localhost:3000",
			shortCode: "abc123",
			expected:  "http://localhost:3000/abc123",
		},
		{
			name:      "base URL with trailing slash",
			baseURL:   "https://short.url",
			shortCode: "xyz789",
			expected:  "https://short.url/xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(kvStore, statsStore, tt.baseURL)
			result := service.GetFullShortURL(tt.shortCode)
			if result != tt.expected {
				t.Errorf("GetFullShortURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestService_Shorten_StoreError(t *testing.T) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	kvStore.putErr = errors.New("storage error")
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	_, err := service.Shorten(ctx, "https://example.com", "test123", 0)
	if err == nil {
		t.Error("Shorten() expected error when store fails")
	}
	if !containsSubstring(err.Error(), "failed to store") {
		t.Errorf("Shorten() error = %q, want error containing 'failed to store'", err.Error())
	}
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkService_Shorten(b *testing.B) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Shorten(ctx, "https://example.com/benchmark", "", 0)
	}
}

func BenchmarkService_Resolve(b *testing.B) {
	ctx := context.Background()
	kvStore := newMockKVStore()
	statsStore := newMockStatsStore()
	service := NewService(kvStore, statsStore, "http://localhost:3000")

	// Create a URL to resolve
	result, _ := service.Shorten(ctx, "https://example.com/benchmark", "bench", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Resolve(ctx, result.ShortCode)
	}
}
