package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
)

// Module implements the analytics consumer module.
// It consumes URL events and tracks analytics data.
type Module struct {
	store  *AnalyticsStore
	logger types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.EventConsumerModule   = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new analytics module.
func NewModule(logger types.Logger) *Module {
	return &Module{
		store:  NewAnalyticsStore(),
		logger: logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "analytics"
}

// RegisterEventConsumers registers event handlers for URL events.
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Register consumer for URLCreated events
	createdDef, ok := registry.GetEventByName("URLCreated", "v1", "shortener")
	if !ok {
		return fmt.Errorf("event URLCreated.v1 not found")
	}
	if err := registry.RegisterEventConsumer(createdDef, m.handleURLCreated, m); err != nil {
		return fmt.Errorf("failed to register URLCreated consumer: %w", err)
	}

	// Register consumer for URLAccessed events
	accessedDef, ok := registry.GetEventByName("URLAccessed", "v1", "shortener")
	if !ok {
		return fmt.Errorf("event URLAccessed.v1 not found")
	}
	if err := registry.RegisterEventConsumer(accessedDef, m.handleURLAccessed, m); err != nil {
		return fmt.Errorf("failed to register URLAccessed consumer: %w", err)
	}

	m.logger.Info("Registered event consumers", "events", []string{"URLCreated.v1", "URLAccessed.v1"})
	return nil
}

// handleURLCreated processes URLCreated events.
func (m *Module) handleURLCreated(_ context.Context, msg *mono.Msg) error {
	var event URLCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		m.logger.Error("Failed to unmarshal URLCreated event", "error", err)
		return nil // Don't retry on unmarshal errors
	}

	m.store.RecordURLCreated(event.ShortCode, event.OriginalURL, event.CreatedAt)
	m.logger.Info("Recorded URL creation",
		"shortCode", event.ShortCode,
		"originalURL", event.OriginalURL)

	return nil
}

// handleURLAccessed processes URLAccessed events.
func (m *Module) handleURLAccessed(_ context.Context, msg *mono.Msg) error {
	var event URLAccessedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		m.logger.Error("Failed to unmarshal URLAccessed event", "error", err)
		return nil // Don't retry on unmarshal errors
	}

	m.store.RecordAccess(AccessLog{
		ShortCode:   event.ShortCode,
		OriginalURL: event.OriginalURL,
		AccessedAt:  event.AccessedAt,
		UserAgent:   event.UserAgent,
		IPAddress:   event.IPAddress,
	})

	m.logger.Debug("Recorded URL access",
		"shortCode", event.ShortCode,
		"accessedAt", event.AccessedAt)

	return nil
}

// Start initializes the analytics module.
func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("Analytics module started")
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("Analytics module stopped")
	return nil
}

// Store returns the analytics store.
func (m *Module) Store() *AnalyticsStore {
	return m.store
}

// RegisterServices registers this module's services in the service container.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// Register get-analytics-summary service
	if err := container.RegisterRequestReplyService("get-analytics-summary", m.handleGetSummary); err != nil {
		return fmt.Errorf("failed to register get-analytics-summary service: %w", err)
	}

	// Register get-analytics-logs service
	if err := container.RegisterRequestReplyService("get-analytics-logs", m.handleGetLogs); err != nil {
		return fmt.Errorf("failed to register get-analytics-logs service: %w", err)
	}

	m.logger.Info("Registered analytics services",
		"services", []string{"get-analytics-summary", "get-analytics-logs"})
	return nil
}

// Service handler functions

// handleGetSummary handles get-analytics-summary service requests.
func (m *Module) handleGetSummary(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	summary := m.store.GetSummary()
	return json.Marshal(summary)
}

// handleGetLogs handles get-analytics-logs service requests.
func (m *Module) handleGetLogs(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Default limit
	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}

	logs := m.store.GetRecentAccessLogs(req.Limit)
	return json.Marshal(logs)
}
