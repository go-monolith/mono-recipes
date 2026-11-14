package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/google/uuid"

	"github.com/example/rate-limiting-middleware/modules/api/contract"
)

// Module implements the API service module. Its services are rate limited by
// the ratelimit middleware, which wraps every request-reply handler when the
// module registers it; the module itself knows nothing about rate limiting.
type Module struct {
	startTime time.Time
	logger    types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
	_ mono.HealthCheckableModule = (*Module)(nil)
)

// NewModule creates a new API module.
func NewModule(logger types.Logger) *Module {
	return &Module{
		logger: logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "api"
}

// Start initializes the API module.
func (m *Module) Start(_ context.Context) error {
	m.startTime = time.Now()
	m.logger.Info("API module started")
	return nil
}

// Stop shuts down the API module.
func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("API module stopped")
	return nil
}

// RegisterServices binds the typed handlers to the module's own contracts.
// Each registration goes through the middleware chain, so the ratelimit
// middleware sees and wraps all three handlers.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.GetData.Register(container, m.handleGetData); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetData.Name, err)
	}
	if err := contract.CreateOrder.Register(container, m.handleCreateOrder); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.CreateOrder.Name, err)
	}
	if err := contract.GetStatus.Register(container, m.handleGetStatus); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetStatus.Name, err)
	}

	m.logger.Info("Registered API services",
		"services", []string{contract.GetData.Subject(), contract.CreateOrder.Subject(), contract.GetStatus.Subject()})
	return nil
}

// Health returns the current health status of the API module.
func (m *Module) Health(_ context.Context) mono.HealthStatus {
	if m.startTime.IsZero() {
		return mono.HealthStatus{
			Healthy: false,
			Message: "not started",
		}
	}
	uptime := time.Since(m.startTime)
	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"uptime": uptime.Round(time.Second).String(),
		},
	}
}

// handleGetData handles the get-data request.
func (m *Module) handleGetData(_ context.Context, _ contract.GetDataRequest, _ *mono.Msg) (contract.DataResponse, error) {
	// Simulate fetching data
	response := contract.DataResponse{
		ID:        uuid.New().String()[:8],
		Name:      "Sample Data",
		Value:     42,
		Timestamp: time.Now(),
	}

	m.logger.Debug("getData request handled", "id", response.ID)
	return response, nil
}

// handleCreateOrder handles the create-order request.
func (m *Module) handleCreateOrder(_ context.Context, req contract.OrderRequest, _ *mono.Msg) (contract.OrderResponse, error) {
	// Validate request
	if req.ProductID == "" {
		return contract.OrderResponse{}, errors.New("product_id is required")
	}
	if req.Quantity <= 0 {
		return contract.OrderResponse{}, errors.New("quantity must be positive")
	}
	if req.Price <= 0 {
		return contract.OrderResponse{}, errors.New("price must be positive")
	}

	// Simulate order creation
	response := contract.OrderResponse{
		OrderID:   uuid.New().String()[:8],
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Total:     float64(req.Quantity) * req.Price,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	m.logger.Debug("createOrder request handled",
		"order_id", response.OrderID,
		"product_id", response.ProductID)
	return response, nil
}

// handleGetStatus handles the get-status request.
func (m *Module) handleGetStatus(_ context.Context, _ contract.GetStatusRequest, _ *mono.Msg) (contract.StatusResponse, error) {
	uptime := time.Since(m.startTime)

	response := contract.StatusResponse{
		Service:   "rate-limiting-demo",
		Status:    "healthy",
		Uptime:    uptime.Round(time.Second).String(),
		Timestamp: time.Now(),
	}

	m.logger.Debug("getStatus request handled", "uptime", response.Uptime)
	return response, nil
}
