package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/url-shortener-demo/modules/shortener"
)

// Module implements the analytics consumer module.
// It consumes URL events and tracks analytics data.
type Module struct {
	store  *AnalyticsStore
	logger types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module              = (*Module)(nil)
	_ mono.EventConsumerModule = (*Module)(nil)
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
	var event shortener.URLCreatedEvent
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
	var event shortener.URLAccessedEvent
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
		"accessedAt", event.AccessedAt.Format(time.RFC3339))

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
