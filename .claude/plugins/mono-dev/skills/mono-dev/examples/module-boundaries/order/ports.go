package order

import (
	"context"

	inventorycontract "example.com/shop/inventory/contract"
)

// InventoryPort is a driven port: what the order domain needs from the outside
// world. The consumer owns it, it speaks the provider's contract DTOs, and it
// says nothing about NATS. Tests substitute a fake for it.
type InventoryPort interface {
	CheckStock(ctx context.Context, req inventorycontract.CheckStockRequest) (inventorycontract.CheckStockResponse, error)
}
