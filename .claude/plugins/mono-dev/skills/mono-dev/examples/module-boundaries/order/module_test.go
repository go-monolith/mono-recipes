package order_test

import (
	"context"
	"testing"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	inventorycontract "example.com/shop/inventory/contract"
	"example.com/shop/order"
	"example.com/shop/order/contract"
)

// standInInventory is a NATS-free stand-in for the inventory module. Its
// service is registered through the provider's own contract.
func standInInventory(t *testing.T, available bool) *monotest.Container {
	t.Helper()
	inventory := monotest.NewContainer("inventory", monotest.NewEventBus())
	err := inventorycontract.CheckStock.Register(inventory,
		func(context.Context, inventorycontract.CheckStockRequest, *types.Msg) (inventorycontract.CheckStockResponse, error) {
			return inventorycontract.CheckStockResponse{Available: available, Stock: 5}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}

// TestOrderModuleConformance is the one-per-module boundary check. It fails,
// with a hint, when the module: has a bad name or dependency list, consumes a
// contract its dependency does not provide, registers on a dependency's
// container, calls an undeclared module's service subject on the bus, never
// uses a declared dependency, publishes an event missing from EmitEvents(), or
// fails Start/Stop.
func TestOrderModuleConformance(t *testing.T) {
	monotest.CheckModule(t, order.NewModule(),
		monotest.WithDependency("inventory", standInInventory(t, true)),
		// Adapters look services up at call time, so drive them:
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			_, err := contract.PlaceOrder.Call(ctx, h.Container(), contract.PlaceOrderRequest{ProductID: "laptop", Quantity: 1})
			return err
		}))
}

// TestPlaceOrderOutOfStock drives the module through the harness.
func TestPlaceOrderOutOfStock(t *testing.T) {
	inventory := standInInventory(t, false)
	h := monotest.New(t, order.NewModule(), monotest.WithDependency("inventory", inventory))
	if err := h.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(context.Background()) }()

	resp := monotest.Call[contract.PlaceOrderRequest, contract.PlaceOrderResponse](t, h, "place-order",
		contract.PlaceOrderRequest{ProductID: "laptop", Quantity: 1})
	if resp.Status != "out_of_stock" {
		t.Errorf("status = %q", resp.Status)
	}
	if inventory.Calls("check-stock") != 1 {
		t.Errorf("inventory consulted %d times", inventory.Calls("check-stock"))
	}
	if v := h.Report().Violations; len(v) != 0 {
		t.Errorf("boundary violations: %+v", v)
	}
}
