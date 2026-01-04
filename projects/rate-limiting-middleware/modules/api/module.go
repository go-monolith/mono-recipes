package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/google/uuid"
)

// Service names for the API module.
const (
	ServiceGetData     = "get-data"
	ServiceCreateOrder = "create-order"
	ServiceGetStatus   = "get-status"
)

// Module implements the API service module.
type Module struct {
	startTime time.Time
	logger    types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                  = (*Module)(nil)
	_ mono.ServiceProviderModule   = (*Module)(nil)
	_ mono.HealthCheckableModule   = (*Module)(nil)
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

// RegisterServices registers the API services.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// Register get-data service
	if err := container.RegisterRequestReplyService(
		ServiceGetData,
		m.handleGetData,
	); err != nil {
		return fmt.Errorf("failed to register %s: %w", ServiceGetData, err)
	}

	// Register create-order service
	if err := container.RegisterRequestReplyService(
		ServiceCreateOrder,
		m.handleCreateOrder,
	); err != nil {
		return fmt.Errorf("failed to register %s: %w", ServiceCreateOrder, err)
	}

	// Register get-status service
	if err := container.RegisterRequestReplyService(
		ServiceGetStatus,
		m.handleGetStatus,
	); err != nil {
		return fmt.Errorf("failed to register %s: %w", ServiceGetStatus, err)
	}

	m.logger.Info("Registered API services",
		"services", []string{ServiceGetData, ServiceCreateOrder, ServiceGetStatus})
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
func (m *Module) handleGetData(_ context.Context, _ *types.Msg) ([]byte, error) {
	// Simulate fetching data
	response := DataResponse{
		ID:        uuid.New().String()[:8],
		Name:      "Sample Data",
		Value:     42,
		Timestamp: time.Now(),
	}

	m.logger.Debug("getData request handled", "id", response.ID)
	return json.Marshal(response)
}

// handleCreateOrder handles the create-order request.
func (m *Module) handleCreateOrder(_ context.Context, msg *types.Msg) ([]byte, error) {
	var req OrderRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Validate request
	if req.ProductID == "" {
		return nil, fmt.Errorf("product_id is required")
	}
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}
	if req.Price <= 0 {
		return nil, fmt.Errorf("price must be positive")
	}

	// Simulate order creation
	response := OrderResponse{
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
	return json.Marshal(response)
}

// handleGetStatus handles the get-status request.
func (m *Module) handleGetStatus(_ context.Context, _ *types.Msg) ([]byte, error) {
	uptime := time.Since(m.startTime)

	response := StatusResponse{
		Service:   "rate-limiting-demo",
		Status:    "healthy",
		Uptime:    uptime.Round(time.Second).String(),
		Timestamp: time.Now(),
	}

	m.logger.Debug("getStatus request handled", "uptime", response.Uptime)
	return json.Marshal(response)
}
