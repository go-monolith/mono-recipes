package analytics

import (
	"sync"
	"time"
)

// AccessLog represents a single access log entry.
type AccessLog struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessedAt  time.Time `json:"accessed_at"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

// URLStats tracks statistics for a single URL.
type URLStats struct {
	ShortCode     string    `json:"short_code"`
	OriginalURL   string    `json:"original_url"`
	TotalAccesses int64     `json:"total_accesses"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
	LastAccessed  time.Time `json:"last_accessed,omitempty"`
}

// DefaultMaxAccessLogs is the default maximum number of access logs to retain.
const DefaultMaxAccessLogs = 10000

// AnalyticsStore provides thread-safe storage for analytics data.
type AnalyticsStore struct {
	mu            sync.RWMutex
	accessLogs    []AccessLog
	urlStats      map[string]*URLStats
	urlsCreated   int64
	maxAccessLogs int
}

// NewAnalyticsStore creates a new analytics store with default limits.
func NewAnalyticsStore() *AnalyticsStore {
	return NewAnalyticsStoreWithLimit(DefaultMaxAccessLogs)
}

// NewAnalyticsStoreWithLimit creates a new analytics store with a custom limit.
func NewAnalyticsStoreWithLimit(maxLogs int) *AnalyticsStore {
	if maxLogs <= 0 {
		maxLogs = DefaultMaxAccessLogs
	}
	return &AnalyticsStore{
		accessLogs:    make([]AccessLog, 0),
		urlStats:      make(map[string]*URLStats),
		maxAccessLogs: maxLogs,
	}
}

// RecordAccess records a URL access event.
func (s *AnalyticsStore) RecordAccess(log AccessLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Append to access logs with size limit (circular buffer behavior)
	s.accessLogs = append(s.accessLogs, log)
	if len(s.accessLogs) > s.maxAccessLogs {
		// Remove oldest entries to stay within limit
		excess := len(s.accessLogs) - s.maxAccessLogs
		s.accessLogs = s.accessLogs[excess:]
	}

	// Update URL stats
	stats, exists := s.urlStats[log.ShortCode]
	if !exists {
		stats = &URLStats{
			ShortCode:   log.ShortCode,
			OriginalURL: log.OriginalURL,
		}
		s.urlStats[log.ShortCode] = stats
	}

	stats.TotalAccesses++
	stats.LastAccessed = log.AccessedAt
}

// RecordURLCreated records a URL creation event.
func (s *AnalyticsStore) RecordURLCreated(shortCode, originalURL string, createdAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urlsCreated++

	// Initialize stats entry
	if _, exists := s.urlStats[shortCode]; !exists {
		s.urlStats[shortCode] = &URLStats{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
			CreatedAt:   createdAt,
		}
	}
}

// GetStats returns statistics for a specific URL.
func (s *AnalyticsStore) GetStats(shortCode string) (*URLStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats, exists := s.urlStats[shortCode]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	copy := *stats
	return &copy, true
}

// GetAllStats returns statistics for all URLs.
func (s *AnalyticsStore) GetAllStats() []URLStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]URLStats, 0, len(s.urlStats))
	for _, stats := range s.urlStats {
		result = append(result, *stats)
	}
	return result
}

// GetRecentAccessLogs returns the most recent access logs.
func (s *AnalyticsStore) GetRecentAccessLogs(limit int) []AccessLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.accessLogs) == 0 {
		return nil
	}

	start := 0
	if len(s.accessLogs) > limit {
		start = len(s.accessLogs) - limit
	}

	result := make([]AccessLog, len(s.accessLogs)-start)
	copy(result, s.accessLogs[start:])
	return result
}

// GetSummary returns an overall analytics summary.
func (s *AnalyticsStore) GetSummary() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalAccesses := int64(0)
	for _, stats := range s.urlStats {
		totalAccesses += stats.TotalAccesses
	}

	return map[string]any{
		"urls_created":   s.urlsCreated,
		"urls_tracked":   len(s.urlStats),
		"total_accesses": totalAccesses,
		"access_logs":    len(s.accessLogs),
	}
}
