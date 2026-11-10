// Package order orchestrates order placement. It imports only the contract
// package of its dependency, never the inventory module package.
package order

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"

	inventorycontract "example.com/shop/inventory/contract"
	"example.com/shop/order/contract"
)

// Module implements the order module.
type Module struct {
	inventory InventoryPort
	eventBus  mono.EventBus
	seq       atomic.Int64
}

var (
	_ mono.DependentModule        = (*Module)(nil)
	_ mono.ServiceProviderModule  = (*Module)(nil)
	_ mono.EventEmitterModule     = (*Module)(nil)
	_ types.ServiceConsumerModule = (*Module)(nil) // no mono.* alias yet
)

// NewModule creates the order module.
func NewModule() *Module { return &Module{} }

func (m *Module) Name() string                  { return "order" }
func (m *Module) Start(_ context.Context) error { return nil }
func (m *Module) Stop(_ context.Context) error  { return nil }

func (m *Module) Dependencies() []string { return []string{"inventory"} }

// ConsumeServices declares the contracts this module calls. The framework
// validates the list when the module starts: the provider must be a declared
// dependency (or a plugin), must have registered the service, and the service
// kind must match. A mismatch fails startup instead of the first call.
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{inventorycontract.CheckStock.ToBase()}
}

// SetDependencyServiceContainer wires the adapter behind the port. The
// container is a read-only view: only look services up on it.
func (m *Module) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == "inventory" {
		m.inventory = inventoryAdapter{container: container}
	}
}

// SetEventBus receives the module-scoped bus. Use it for events; never for a
// raw request on another module's services.<module>.<service> subject.
func (m *Module) SetEventBus(bus mono.EventBus) { m.eventBus = bus }

func (m *Module) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{contract.OrderPlacedV1.ToBase()}
}

func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	return contract.PlaceOrder.Register(container, m.placeOrder)
}

// placeOrder is domain logic: it talks to ports, not to containers or subjects.
func (m *Module) placeOrder(ctx context.Context, req contract.PlaceOrderRequest, _ *mono.Msg) (contract.PlaceOrderResponse, error) {
	stock, err := m.inventory.CheckStock(ctx, inventorycontract.CheckStockRequest{ProductID: req.ProductID, Quantity: req.Quantity})
	if err != nil {
		return contract.PlaceOrderResponse{}, err
	}
	orderID := fmt.Sprintf("order-%d", m.seq.Add(1))
	if !stock.Available {
		return contract.PlaceOrderResponse{OrderID: orderID, Status: "out_of_stock"}, nil
	}
	event := contract.OrderPlacedEvent{OrderID: orderID, ProductID: req.ProductID}
	if err := contract.OrderPlacedV1.Publish(m.eventBus, event, nil); err != nil {
		slog.Warn("order placed event not published", "order_id", orderID, "error", err) // best effort
	}
	return contract.PlaceOrderResponse{OrderID: orderID, Status: "placed"}, nil
}
