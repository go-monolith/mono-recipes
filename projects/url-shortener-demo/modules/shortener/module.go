package shortener

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"
	"github.com/go-monolith/mono/pkg/types"
)

// Module implements the URL shortener module using the kv-jetstream plugin.
type Module struct {
	kv       *kvjetstream.PluginModule
	bucket   kvjetstream.KVStoragePort
	service  *Service
	eventBus mono.EventBus
	baseURL  string
	logger   types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module              = (*Module)(nil)
	_ mono.UsePluginModule     = (*Module)(nil)
	_ mono.EventBusAwareModule = (*Module)(nil)
	_ mono.EventEmitterModule  = (*Module)(nil)
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
	return "shortener"
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

// SetEventBus receives the EventBus from the framework.
func (m *Module) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents declares the events this module can emit.
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		URLCreatedV1.ToBase(),
		URLAccessedV1.ToBase(),
	}
}

// Start initializes the module and its service.
func (m *Module) Start(ctx context.Context) error {
	if m.kv == nil {
		return fmt.Errorf("required plugin 'kv' not registered")
	}

	// Get the urls bucket from the KV plugin
	m.bucket = m.kv.Bucket("urls")
	if m.bucket == nil {
		return fmt.Errorf("bucket 'urls' not found in KV plugin")
	}

	// Create the service
	var err error
	m.service, err = NewService(m.bucket, m.baseURL)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	m.logger.Info("URL shortener module started", "baseURL", m.baseURL)
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("URL shortener module stopped")
	return nil
}

// Service returns the shortener service instance.
func (m *Module) Service() *Service {
	return m.service
}

// EventBus returns the event bus for publishing events.
func (m *Module) EventBus() mono.EventBus {
	return m.eventBus
}

// PublishURLCreated publishes a URLCreated event.
func (m *Module) PublishURLCreated(event URLCreatedEvent) error {
	return URLCreatedV1.Publish(m.eventBus, event, nil)
}

// PublishURLAccessed publishes a URLAccessed event.
func (m *Module) PublishURLAccessed(event URLAccessedEvent) error {
	return URLAccessedV1.Publish(m.eventBus, event, nil)
}

// RegisterServices registers this module's services in the service container.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// Register shorten-url service
	if err := container.RegisterRequestReplyService("shorten-url", m.handleShortenURL); err != nil {
		return fmt.Errorf("failed to register shorten-url service: %w", err)
	}

	// Register resolve-url service
	if err := container.RegisterRequestReplyService("resolve-url", m.handleResolveURL); err != nil {
		return fmt.Errorf("failed to register resolve-url service: %w", err)
	}

	// Register get-stats service
	if err := container.RegisterRequestReplyService("get-stats", m.handleGetStats); err != nil {
		return fmt.Errorf("failed to register get-stats service: %w", err)
	}

	// Register list-urls service
	if err := container.RegisterRequestReplyService("list-urls", m.handleListURLs); err != nil {
		return fmt.Errorf("failed to register list-urls service: %w", err)
	}

	// Register delete-url service
	if err := container.RegisterRequestReplyService("delete-url", m.handleDeleteURL); err != nil {
		return fmt.Errorf("failed to register delete-url service: %w", err)
	}

	m.logger.Info("Registered shortener services",
		"services", []string{"shorten-url", "resolve-url", "get-stats", "list-urls", "delete-url"})
	return nil
}

// Service handler functions

// handleShortenURL handles shorten-url service requests.
func (m *Module) handleShortenURL(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req ShortenRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	result, err := m.service.ShortenURL(ctx, req)
	if err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	// Publish URLCreated event internally
	if publishErr := m.PublishURLCreated(URLCreatedEvent{
		ShortCode:   result.ShortCode,
		OriginalURL: result.OriginalURL,
		CreatedAt:   result.CreatedAt,
		TTLSeconds:  req.TTLSeconds,
	}); publishErr != nil {
		m.logger.Warn("Failed to publish URLCreated event",
			"shortCode", result.ShortCode,
			"error", publishErr)
	}

	respData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return respData, nil
}

// handleResolveURL handles resolve-url service requests.
func (m *Module) handleResolveURL(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req struct {
		ShortCode string `json:"short_code"`
		UserAgent string `json:"user_agent,omitempty"`
		IPAddress string `json:"ip_address,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	originalURL, err := m.service.ResolveAndTrack(ctx, req.ShortCode)
	if err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	// Publish URLAccessed event internally
	if publishErr := m.PublishURLAccessed(URLAccessedEvent{
		ShortCode:   req.ShortCode,
		OriginalURL: originalURL,
		AccessedAt:  time.Now(),
		UserAgent:   req.UserAgent,
		IPAddress:   req.IPAddress,
	}); publishErr != nil {
		m.logger.Debug("Failed to publish URLAccessed event",
			"shortCode", req.ShortCode,
			"error", publishErr)
	}

	response := struct {
		OriginalURL string `json:"original_url"`
	}{
		OriginalURL: originalURL,
	}
	respData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return respData, nil
}

// handleGetStats handles get-stats service requests.
func (m *Module) handleGetStats(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	stats, err := m.service.GetStats(ctx, req.ShortCode)
	if err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	respData, err := json.Marshal(stats)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return respData, nil
}

// handleListURLs handles list-urls service requests.
func (m *Module) handleListURLs(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	urls, err := m.service.ListURLs(ctx)
	if err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	respData, err := json.Marshal(urls)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return respData, nil
}

// handleDeleteURL handles delete-url service requests.
func (m *Module) handleDeleteURL(ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	err := m.service.DeleteURL(ctx, req.ShortCode)
	if err != nil {
		errResp, marshalErr := json.Marshal(map[string]string{
			"error": err.Error(),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal error response: %w", marshalErr)
		}
		return errResp, nil
	}

	response := struct {
		Message   string `json:"message"`
		ShortCode string `json:"short_code"`
	}{
		Message:   "URL deleted successfully",
		ShortCode: req.ShortCode,
	}
	respData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	return respData, nil
}
