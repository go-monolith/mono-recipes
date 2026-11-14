// Package analytics consumes the shortener's URL events and serves analytics
// over its own contract. It imports only shortener/contract, never the
// shortener module package.
package analytics

import (
	"context"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/url-shortener-demo/modules/analytics/contract"
	shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"
)

// Module implements the analytics module.
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
	return contract.ModuleName
}

// RegisterEventConsumers subscribes to the shortener's events through the
// definitions published in shortener/contract: no name/version strings, and
// the payload arrives already decoded.
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
	if err := helper.RegisterTypedEventConsumer(registry, shortenercontract.URLCreatedV1, m.handleURLCreated, m); err != nil {
		return fmt.Errorf("failed to register URLCreated consumer: %w", err)
	}
	if err := helper.RegisterTypedEventConsumer(registry, shortenercontract.URLAccessedV1, m.handleURLAccessed, m); err != nil {
		return fmt.Errorf("failed to register URLAccessed consumer: %w", err)
	}

	m.logger.Info("Registered event consumers",
		"events", []string{shortenercontract.URLCreatedV1.Subject, shortenercontract.URLAccessedV1.Subject})
	return nil
}

// handleURLCreated processes URLCreated events.
func (m *Module) handleURLCreated(_ context.Context, event shortenercontract.URLCreatedEvent, _ *mono.Msg) error {
	m.store.RecordURLCreated(event.ShortCode, event.OriginalURL, event.CreatedAt)
	m.logger.Info("Recorded URL creation",
		"shortCode", event.ShortCode,
		"originalURL", event.OriginalURL)
	return nil
}

// handleURLAccessed processes URLAccessed events.
func (m *Module) handleURLAccessed(_ context.Context, event shortenercontract.URLAccessedEvent, _ *mono.Msg) error {
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
func (m *Module) Start(_ context.Context) error {
	m.logger.Info("Analytics module started")
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("Analytics module stopped")
	return nil
}

// RegisterServices is the driving adapter: it binds the typed handlers to the
// module's own contracts.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.GetSummary.Register(container, m.handleGetSummary); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetSummary.Name, err)
	}
	if err := contract.GetLogs.Register(container, m.handleGetLogs); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetLogs.Name, err)
	}

	m.logger.Info("Registered analytics services",
		"services", []string{contract.GetSummary.Name, contract.GetLogs.Name})
	return nil
}

// handleGetSummary handles get-analytics-summary service requests.
func (m *Module) handleGetSummary(_ context.Context, _ contract.SummaryRequest, _ *mono.Msg) (contract.Summary, error) {
	return m.store.GetSummary(), nil
}

// handleGetLogs handles get-analytics-logs service requests.
func (m *Module) handleGetLogs(_ context.Context, req contract.LogsRequest, _ *mono.Msg) ([]contract.AccessLog, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	return m.store.GetRecentAccessLogs(limit), nil
}
