package product_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-monolith/mono/monotest"

	"github.com/example/gorm-sqlite-demo/modules/product"
	"github.com/example/gorm-sqlite-demo/modules/product/contract"
)

// newModule returns a product module backed by a fresh SQLite file.
func newModule(t *testing.T) *product.ProductModule {
	t.Helper()
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "products.db"))
	return product.NewModule()
}

// TestProductModuleConformance is the one-per-module boundary check: it starts
// the module on a NATS-free harness, drives every contract it provides through
// the contract itself, and fails on any Ports and Adapters rule the module
// breaks (bad name, no services, boundary violations, Start/Stop failures).
func TestProductModuleConformance(t *testing.T) {
	monotest.CheckModule(t, newModule(t),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseCRUD(ctx, h)
		}))
}

// exerciseCRUD calls every product contract once, in CRUD order.
func exerciseCRUD(ctx context.Context, h *monotest.Harness) error {
	c := h.Container()

	created, err := contract.CreateProduct.Call(ctx, c, contract.CreateProductRequest{
		Name: "Widget", Description: "A widget", Price: 9.99, Stock: 3,
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	got, err := contract.GetProduct.Call(ctx, c, contract.GetProductRequest{ID: created.ID})
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if got.Name != "Widget" {
		return fmt.Errorf("get: name = %q, want Widget", got.Name)
	}

	list, err := contract.ListProducts.Call(ctx, c, contract.ListProductsRequest{})
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if list.Total != 1 {
		return fmt.Errorf("list: total = %d, want 1", list.Total)
	}

	price := 12.5
	updated, err := contract.UpdateProduct.Call(ctx, c, contract.UpdateProductRequest{ID: created.ID, Price: &price})
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if updated.Price != price {
		return fmt.Errorf("update: price = %v, want %v", updated.Price, price)
	}

	deleted, err := contract.DeleteProduct.Call(ctx, c, contract.DeleteProductRequest{ID: created.ID})
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if !deleted.Deleted {
		return fmt.Errorf("delete: deleted = false")
	}

	if _, err := contract.GetProduct.Call(ctx, c, contract.GetProductRequest{ID: created.ID}); err == nil {
		return fmt.Errorf("get after delete: expected an error")
	}
	return nil
}

// TestProductModuleConcurrentRequests guards the SQLite settings in Start:
// request-reply handlers run concurrently on a worker pool, and concurrent
// writes must queue rather than fail with "database is locked".
func TestProductModuleConcurrentRequests(t *testing.T) {
	h := monotest.New(t, newModule(t))
	ctx := context.Background()
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Go(func() {
			_, err := contract.CreateProduct.Call(ctx, h.Container(), contract.CreateProductRequest{
				Name: fmt.Sprintf("p-%d", i), Price: 1, Stock: 1,
			})
			if err == nil {
				_, err = contract.ListProducts.Call(ctx, h.Container(), contract.ListProductsRequest{})
			}
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent requests: %v", err)
		}
	}

	list, err := contract.ListProducts.Call(ctx, h.Container(), contract.ListProductsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != n {
		t.Errorf("total = %d, want %d", list.Total, n)
	}
}

// TestCreateProductValidation checks that handler errors travel back through
// the contract.
func TestCreateProductValidation(t *testing.T) {
	h := monotest.New(t, newModule(t))
	ctx := context.Background()
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	if _, err := contract.CreateProduct.Call(ctx, h.Container(), contract.CreateProductRequest{Price: 1}); err == nil {
		t.Error("expected an error for a missing name")
	}
}
