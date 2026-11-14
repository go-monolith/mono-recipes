// Package shortener holds the domain logic and the driving adapter of the
// shortener module. Other modules import shortener/contract, not this package.
package shortener

import (
	"context"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"

	"github.com/example/url-shortener-demo/modules/shortener/contract"
)

// Module implements the URL shortener module using the kv-jetstream plugin.
type Module struct {
	kv       *kvjetstream.PluginModule
	service  *Service
	eventBus mono.EventBus
	baseURL  string
	logger   types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.UsePluginModule       = (*Module)(nil)
	_ mono.EventBusAwareModule   = (*Module)(nil)
	_ mono.EventEmitterModule    = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new URL shortener module.
func NewModule(baseURL string, logger types.Logger) *Module {
	return &Module{
		baseURL: baseURL,
		logger:  logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return contract.ModuleName
}

// SetPlugin receives the KV plugin from the framework.
// This is called before Start() when the module implements UsePluginModule.
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "kv" {
		kv, ok := plugin.(*kvjetstream.PluginModule)
		if !ok {
			m.logger.Error("Invalid plugin type for kv",
				"alias", alias,
				"expected", "*kvjetstream.PluginModule")
			return
		}
		m.kv = kv
		m.logger.Info("Received KV plugin", "alias", alias)
	}
}

// SetEventBus receives the module-scoped EventBus from the framework.
func (m *Module) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents declares the events this module can emit. The definitions live
// in the contract package so consumers (analytics) can subscribe without
// importing this package.
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		contract.URLCreatedV1.ToBase(),
		contract.URLAccessedV1.ToBase(),
	}
}

// Start initializes the module and its service.
func (m *Module) Start(_ context.Context) error {
	if m.kv == nil {
		return fmt.Errorf("required plugin 'kv' not registered")
	}

	// Get the urls bucket from the KV plugin
	bucket := m.kv.Bucket("urls")
	if bucket == nil {
		return fmt.Errorf("bucket 'urls' not found in KV plugin")
	}

	// Create the service
	var err error
	m.service, err = NewService(bucket, m.baseURL)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	m.logger.Info("URL shortener module started", "baseURL", m.baseURL)
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("URL shortener module stopped")
	return nil
}

// RegisterServices is the driving adapter: it binds the typed handlers to the
// module's own contracts. No service-name strings, no manual JSON.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.ShortenURL.Register(container, m.handleShortenURL); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.ShortenURL.Name, err)
	}
	if err := contract.ResolveURL.Register(container, m.handleResolveURL); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.ResolveURL.Name, err)
	}
	if err := contract.GetStats.Register(container, m.handleGetStats); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetStats.Name, err)
	}
	if err := contract.ListURLs.Register(container, m.handleListURLs); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.ListURLs.Name, err)
	}
	if err := contract.DeleteURL.Register(container, m.handleDeleteURL); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.DeleteURL.Name, err)
	}

	m.logger.Info("Registered shortener services",
		"services", []string{
			contract.ShortenURL.Name, contract.ResolveURL.Name, contract.GetStats.Name,
			contract.ListURLs.Name, contract.DeleteURL.Name,
		})
	return nil
}

// Service handlers. An error returned here travels back to the caller as a
// RemoteError carrying the error message; consumers map it back to the
// contract's sentinel errors.

// handleShortenURL handles shorten-url service requests.
func (m *Module) handleShortenURL(ctx context.Context, req contract.ShortenRequest, _ *mono.Msg) (contract.ShortenResponse, error) {
	result, err := m.service.ShortenURL(ctx, req)
	if err != nil {
		return contract.ShortenResponse{}, err
	}

	event := contract.URLCreatedEvent{
		ShortCode:   result.ShortCode,
		OriginalURL: result.OriginalURL,
		CreatedAt:   result.CreatedAt,
		TTLSeconds:  req.TTLSeconds,
	}
	if err := contract.URLCreatedV1.Publish(m.eventBus, event, nil); err != nil {
		m.logger.Warn("Failed to publish URLCreated event",
			"shortCode", result.ShortCode,
			"error", err)
	}

	return *result, nil
}

// handleResolveURL handles resolve-url service requests.
func (m *Module) handleResolveURL(ctx context.Context, req contract.ResolveRequest, _ *mono.Msg) (contract.ResolveResponse, error) {
	originalURL, err := m.service.ResolveAndTrack(ctx, req.ShortCode)
	if err != nil {
		return contract.ResolveResponse{}, err
	}

	event := contract.URLAccessedEvent{
		ShortCode:   req.ShortCode,
		OriginalURL: originalURL,
		AccessedAt:  time.Now(),
		UserAgent:   req.UserAgent,
		IPAddress:   req.IPAddress,
	}
	if err := contract.URLAccessedV1.Publish(m.eventBus, event, nil); err != nil {
		m.logger.Debug("Failed to publish URLAccessed event",
			"shortCode", req.ShortCode,
			"error", err)
	}

	return contract.ResolveResponse{OriginalURL: originalURL}, nil
}

// handleGetStats handles get-stats service requests.
func (m *Module) handleGetStats(ctx context.Context, req contract.ShortCodeRequest, _ *mono.Msg) (contract.StatsResponse, error) {
	stats, err := m.service.GetStats(ctx, req.ShortCode)
	if err != nil {
		return contract.StatsResponse{}, err
	}
	return *stats, nil
}

// handleListURLs handles list-urls service requests.
func (m *Module) handleListURLs(ctx context.Context, _ contract.ListURLsRequest, _ *mono.Msg) ([]contract.URLEntry, error) {
	return m.service.ListURLs(ctx)
}

// handleDeleteURL handles delete-url service requests.
func (m *Module) handleDeleteURL(ctx context.Context, req contract.ShortCodeRequest, _ *mono.Msg) (contract.DeleteResponse, error) {
	if err := m.service.DeleteURL(ctx, req.ShortCode); err != nil {
		return contract.DeleteResponse{}, err
	}
	return contract.DeleteResponse{
		Message:   "URL deleted successfully",
		ShortCode: req.ShortCode,
	}, nil
}
