package nats

import (
	"context"
	"errors"
	"log"

	"github.com/go-monolith/mono"
)

// Module provides NATS JetStream as a mono module.
type Module struct {
	client *Client
	config Config
}

// NewModule creates a new NATS module with default configuration.
func NewModule(natsURL string) *Module {
	cfg := DefaultConfig()
	cfg.URL = natsURL
	return &Module{
		config: cfg,
	}
}

// NewModuleWithConfig creates a new NATS module with custom configuration.
func NewModuleWithConfig(cfg Config) *Module {
	return &Module{
		config: cfg,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "nats"
}

// Init initializes the NATS client.
func (m *Module) Init(_ mono.ServiceContainer) error {
	m.client = NewClient(m.config)
	return nil
}

// Start connects to NATS and creates the consumer.
func (m *Module) Start(ctx context.Context) error {
	if err := m.client.Connect(ctx); err != nil {
		return err
	}

	if err := m.client.CreateConsumer(ctx, m.config); err != nil {
		return err
	}

	log.Println("[nats] Module started")
	return nil
}

// Stop closes the NATS connection.
func (m *Module) Stop(_ context.Context) error {
	if m.client != nil {
		return m.client.Close()
	}
	log.Println("[nats] Module stopped")
	return nil
}

// GetClient returns the NATS client.
func (m *Module) GetClient() *Client {
	return m.client
}

// HealthCheck verifies the NATS connection is healthy.
func (m *Module) HealthCheck(_ context.Context) error {
	if m.client == nil || !m.client.IsConnected() {
		return ErrNotConnected
	}
	return nil
}

// ErrNotConnected is returned when NATS is not connected.
var ErrNotConnected = errors.New("nats not connected")
