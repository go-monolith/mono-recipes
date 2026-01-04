package shortener

import (
	"context"
	"fmt"

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
	_ mono.Module            = (*Module)(nil)
	_ mono.UsePluginModule   = (*Module)(nil)
	_ mono.EventBusAwareModule = (*Module)(nil)
	_ mono.EventEmitterModule  = (*Module)(nil)
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
