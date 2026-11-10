// Package inventory holds the domain logic and the driving adapter of the
// inventory module. Other modules import inventory/contract, not this package.
package inventory

import (
	"context"
	"errors"
	"sync"

	"github.com/go-monolith/mono"

	"example.com/shop/inventory/contract"
)

// Module implements the inventory module.
type Module struct {
	mu    sync.RWMutex
	stock map[string]int
}

var _ mono.ServiceProviderModule = (*Module)(nil)

// NewModule creates the inventory module with some stock.
func NewModule() *Module {
	return &Module{stock: map[string]int{"laptop": 10, "mouse": 50}}
}

func (m *Module) Name() string                  { return "inventory" }
func (m *Module) Start(_ context.Context) error { return nil }
func (m *Module) Stop(_ context.Context) error  { return nil }

// RegisterServices is the driving adapter: it binds the typed domain handler
// to the module's own contract. No service-name string, no manual JSON.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	return contract.CheckStock.Register(container, m.checkStock)
}

// checkStock is pure domain logic and can be unit-tested by calling it directly.
func (m *Module) checkStock(_ context.Context, req contract.CheckStockRequest, _ *mono.Msg) (contract.CheckStockResponse, error) {
	if req.ProductID == "" || req.Quantity <= 0 {
		return contract.CheckStockResponse{}, errors.New("product_id and a positive quantity are required")
	}
	m.mu.RLock()
	stock := m.stock[req.ProductID]
	m.mu.RUnlock()
	return contract.CheckStockResponse{Available: stock >= req.Quantity, Stock: stock}, nil
}
