package analytics

import (
	"testing"
	"time"
)

func TestAnalyticsStore_RecordAccess(t *testing.T) {
	store := NewAnalyticsStore()

	log := AccessLog{
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		AccessedAt:  time.Now(),
		UserAgent:   "Mozilla/5.0",
		IPAddress:   "192.168.1.1",
	}

	store.RecordAccess(log)

	stats, exists := store.GetStats("abc123")
	if !exists {
		t.Fatal("Expected stats to exist after recording access")
	}

	if stats.TotalAccesses != 1 {
		t.Errorf("Expected TotalAccesses = 1, got %d", stats.TotalAccesses)
	}

	if stats.ShortCode != "abc123" {
		t.Errorf("Expected ShortCode = 'abc123', got %q", stats.ShortCode)
	}
}

func TestAnalyticsStore_RecordMultipleAccesses(t *testing.T) {
	store := NewAnalyticsStore()

	for i := 0; i < 5; i++ {
		store.RecordAccess(AccessLog{
			ShortCode:   "abc123",
			OriginalURL: "https://example.com",
			AccessedAt:  time.Now(),
		})
	}

	stats, exists := store.GetStats("abc123")
	if !exists {
		t.Fatal("Expected stats to exist")
	}

	if stats.TotalAccesses != 5 {
		t.Errorf("Expected TotalAccesses = 5, got %d", stats.TotalAccesses)
	}
}

func TestAnalyticsStore_RecordURLCreated(t *testing.T) {
	store := NewAnalyticsStore()
	now := time.Now()

	store.RecordURLCreated("xyz789", "https://test.com", now)

	stats, exists := store.GetStats("xyz789")
	if !exists {
		t.Fatal("Expected stats to exist after URL creation")
	}

	if stats.ShortCode != "xyz789" {
		t.Errorf("Expected ShortCode = 'xyz789', got %q", stats.ShortCode)
	}

	if stats.OriginalURL != "https://test.com" {
		t.Errorf("Expected OriginalURL = 'https://test.com', got %q", stats.OriginalURL)
	}
}

func TestAnalyticsStore_GetSummary(t *testing.T) {
	store := NewAnalyticsStore()

	// Create some URLs
	store.RecordURLCreated("url1", "https://example1.com", time.Now())
	store.RecordURLCreated("url2", "https://example2.com", time.Now())

	// Record some accesses
	store.RecordAccess(AccessLog{ShortCode: "url1", AccessedAt: time.Now()})
	store.RecordAccess(AccessLog{ShortCode: "url1", AccessedAt: time.Now()})
	store.RecordAccess(AccessLog{ShortCode: "url2", AccessedAt: time.Now()})

	summary := store.GetSummary()

	if summary["urls_created"].(int64) != 2 {
		t.Errorf("Expected urls_created = 2, got %v", summary["urls_created"])
	}

	if summary["total_accesses"].(int64) != 3 {
		t.Errorf("Expected total_accesses = 3, got %v", summary["total_accesses"])
	}

	if summary["access_logs"].(int) != 3 {
		t.Errorf("Expected access_logs = 3, got %v", summary["access_logs"])
	}
}

func TestAnalyticsStore_GetRecentAccessLogs(t *testing.T) {
	store := NewAnalyticsStore()

	// Record 10 accesses
	for i := 0; i < 10; i++ {
		store.RecordAccess(AccessLog{
			ShortCode:  "test",
			AccessedAt: time.Now(),
		})
	}

	// Get only last 5
	logs := store.GetRecentAccessLogs(5)
	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}

	// Get all when limit exceeds count
	logs = store.GetRecentAccessLogs(100)
	if len(logs) != 10 {
		t.Errorf("Expected 10 logs, got %d", len(logs))
	}
}

func TestAnalyticsStore_GetAllStats(t *testing.T) {
	store := NewAnalyticsStore()

	store.RecordURLCreated("url1", "https://example1.com", time.Now())
	store.RecordURLCreated("url2", "https://example2.com", time.Now())
	store.RecordURLCreated("url3", "https://example3.com", time.Now())

	allStats := store.GetAllStats()
	if len(allStats) != 3 {
		t.Errorf("Expected 3 stats entries, got %d", len(allStats))
	}
}

func TestAnalyticsStore_NonExistentStats(t *testing.T) {
	store := NewAnalyticsStore()

	stats, exists := store.GetStats("nonexistent")
	if exists {
		t.Error("Expected exists = false for nonexistent short code")
	}
	if stats != nil {
		t.Error("Expected stats = nil for nonexistent short code")
	}
}
