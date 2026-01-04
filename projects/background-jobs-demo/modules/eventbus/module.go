package eventbus

import (
	"context"
	"log"

	"github.com/go-monolith/mono"
)

// Module provides the EventBus as a mono module.
type Module struct {
	eventBus *EventBus
}

// NewModule creates a new EventBus module.
func NewModule() *Module {
	return &Module{}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "eventbus"
}

// Init initializes the EventBus.
func (m *Module) Init(_ mono.ServiceContainer) error {
	m.eventBus = New()
	log.Println("[eventbus] EventBus initialized")
	return nil
}

// Start starts the module.
func (m *Module) Start(_ context.Context) error {
	log.Println("[eventbus] Module started")
	return nil
}

// Stop stops the module.
func (m *Module) Stop(_ context.Context) error {
	log.Println("[eventbus] Module stopped")
	return nil
}

// GetEventBus returns the EventBus instance.
func (m *Module) GetEventBus() *EventBus {
	return m.eventBus
}
