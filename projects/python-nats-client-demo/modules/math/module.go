package math

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
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

// RegisterServices registers request-reply services in the service container.
// The framework automatically prefixes service names with "services.<module>."
// so "calculate" becomes "services.math.calculate" in the NATS subject.
func (m *MathModule) RegisterServices(container mono.ServiceContainer) error {
	if err := helper.RegisterTypedRequestReplyService(
		container, "calculate", json.Unmarshal, json.Marshal, m.calculate,
	); err != nil {
		return fmt.Errorf("failed to register calculate service: %w", err)
	}

	log.Printf("[math] Registered services: services.math.calculate")
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
