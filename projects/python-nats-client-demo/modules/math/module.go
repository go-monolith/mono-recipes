package math

import (
	"context"
	"fmt"
	"log"

	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/math/contract"
)

// MathModule provides math calculation services via RequestReplyService.
type MathModule struct{}

// Compile-time interface checks.
var (
	_ mono.Module                = (*MathModule)(nil)
	_ mono.ServiceProviderModule = (*MathModule)(nil)
)

// NewModule creates a new MathModule.
func NewModule() *MathModule {
	return &MathModule{}
}

// Name returns the module name.
func (m *MathModule) Name() string {
	return "math"
}

// RegisterServices binds the typed handler to the module's own contract.
// The contract fixes the NATS subject (services.math.calculate) and the JSON
// encoding the Python MathClient relies on.
func (m *MathModule) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.Calculate.Register(container, m.calculate); err != nil {
		return fmt.Errorf("failed to register calculate service: %w", err)
	}

	log.Printf("[math] Registered services: %s", contract.Calculate.Subject())
	return nil
}

// Start initializes the math module.
func (m *MathModule) Start(_ context.Context) error {
	log.Println("[math] Module started successfully")
	return nil
}

// Stop gracefully stops the math module.
func (m *MathModule) Stop(_ context.Context) error {
	log.Println("[math] Module stopped")
	return nil
}
