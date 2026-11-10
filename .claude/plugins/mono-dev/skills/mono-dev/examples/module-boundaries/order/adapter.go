package order

import (
	"context"
	"fmt"

	"github.com/go-monolith/mono"

	inventorycontract "example.com/shop/inventory/contract"
)

// inventoryAdapter is the driven adapter: it implements InventoryPort by
// calling the provider's contract with the container received in
// SetDependencyServiceContainer. It is the only place in the module that knows
// the call crosses a module boundary.
type inventoryAdapter struct{ container mono.ServiceContainer }

var _ InventoryPort = inventoryAdapter{}

func (a inventoryAdapter) CheckStock(ctx context.Context, req inventorycontract.CheckStockRequest) (inventorycontract.CheckStockResponse, error) {
	resp, err := inventorycontract.CheckStock.Call(ctx, a.container, req)
	if err != nil {
		return inventorycontract.CheckStockResponse{}, fmt.Errorf("inventory check failed: %w", err)
	}
	return resp, nil
}
