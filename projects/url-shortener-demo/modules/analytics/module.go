package analytics

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/example/url-shortener-demo/events"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// AccessLog represents a logged URL access event.
type AccessLog struct {
	ShortCode  string    `json:"short_code"`
	AccessedAt time.Time `json:"accessed_at"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Referer    string    `json:"referer,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
}

// CreationLog represents a logged URL creation event.
type CreationLog struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// maxLogEntries is the maximum number of log entries to keep in memory.
// Older entries are discarded when this limit is reached (circular buffer behavior).
const maxLogEntries = 10000

// AnalyticsModule handles URL analytics as an event consumer.
// It subscribes to URL events using the EventConsumerModule interface.
// Note: Uses bounded in-memory storage with circular buffer to prevent memory exhaustion.
type AnalyticsModule struct {
	accessLogs   []AccessLog
	creationLogs []CreationLog
	mu           sync.RWMutex
}

// Compile-time interface checks.
var _ mono.Module = (*AnalyticsModule)(nil)
var _ mono.EventConsumerModule = (*AnalyticsModule)(nil)

// NewModule creates a new AnalyticsModule.
func NewModule() *AnalyticsModule {
	return &AnalyticsModule{
		accessLogs:   make([]AccessLog, 0),
		creationLogs: make([]CreationLog, 0),
	}
}

// Name returns the module name.
func (m *AnalyticsModule) Name() string {
	return "analytics"
}

// RegisterEventConsumers registers event consumers for this module.
func (m *AnalyticsModule) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Subscribe to URLCreated events
	if err := helper.RegisterTypedEventConsumer(registry, events.URLCreatedV1, m.handleURLCreated, m); err != nil {
		return fmt.Errorf("failed to register URLCreated consumer: %w", err)
	}

	// Subscribe to URLAccessed events
	if err := helper.RegisterTypedEventConsumer(registry, events.URLAccessedV1, m.handleURLAccessed, m); err != nil {
		return fmt.Errorf("failed to register URLAccessed consumer: %w", err)
	}

	log.Printf("[analytics] Registered event consumers: URLCreated, URLAccessed")
	return nil
}

// handleURLCreated handles URLCreated events.
func (m *AnalyticsModule) handleURLCreated(_ context.Context, event events.URLCreatedEvent, _ *mono.Msg) error {
	log.Printf("[analytics] URL created: %s -> %s", event.ShortCode, event.OriginalURL)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.creationLogs = append(m.creationLogs, CreationLog{
		ShortCode:   event.ShortCode,
		OriginalURL: event.OriginalURL,
		CreatedAt:   event.CreatedAt,
	})

	// Enforce max size (circular buffer behavior)
	if len(m.creationLogs) > maxLogEntries {
		m.creationLogs = m.creationLogs[len(m.creationLogs)-maxLogEntries:]
	}

	// In a real system: send to analytics platform, store in time-series DB, etc.
	return nil
}

// handleURLAccessed handles URLAccessed events.
func (m *AnalyticsModule) handleURLAccessed(_ context.Context, event events.URLAccessedEvent, _ *mono.Msg) error {
	log.Printf("[analytics] URL accessed: %s at %s (UA: %s)", event.ShortCode, event.AccessedAt.Format(time.RFC3339), event.UserAgent)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.accessLogs = append(m.accessLogs, AccessLog{
		ShortCode:  event.ShortCode,
		AccessedAt: event.AccessedAt,
		UserAgent:  event.UserAgent,
		Referer:    event.Referer,
		IPAddress:  event.IPAddress,
	})

	// Enforce max size (circular buffer behavior)
	if len(m.accessLogs) > maxLogEntries {
		m.accessLogs = m.accessLogs[len(m.accessLogs)-maxLogEntries:]
	}

	// In a real system: track click-through rates, geographic data, etc.
	return nil
}

// GetAccessLogs returns a copy of all logged access events.
func (m *AnalyticsModule) GetAccessLogs() []AccessLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]AccessLog, len(m.accessLogs))
	copy(result, m.accessLogs)
	return result
}

// GetCreationLogs returns a copy of all logged creation events.
func (m *AnalyticsModule) GetCreationLogs() []CreationLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]CreationLog, len(m.creationLogs))
	copy(result, m.creationLogs)
	return result
}

// GetTotalAccessCount returns the total number of URL accesses.
func (m *AnalyticsModule) GetTotalAccessCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.accessLogs)
}

// GetTotalURLsCreated returns the total number of URLs created.
func (m *AnalyticsModule) GetTotalURLsCreated() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.creationLogs)
}

// Start initializes the module.
func (m *AnalyticsModule) Start(_ context.Context) error {
	log.Println("[analytics] Module started - listening for URL events")
	return nil
}

// Stop shuts down the module.
func (m *AnalyticsModule) Stop(_ context.Context) error {
	m.mu.RLock()
	accessCount := len(m.accessLogs)
	creationCount := len(m.creationLogs)
	m.mu.RUnlock()

	log.Printf("[analytics] Module stopped - logged %d creations, %d accesses", creationCount, accessCount)
	return nil
}
