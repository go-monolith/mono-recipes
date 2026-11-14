package product_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/example/redis-caching-demo/modules/cache"
	"github.com/example/redis-caching-demo/modules/product"
	"github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/storage"
)

// memStorage is an in-memory storage.Storage, so the real cache plugin can run
// in tests without Redis.
type memStorage struct {
	mu   sync.Mutex
	data map[string][]byte
}

var _ storage.Storage = (*memStorage)(nil)

func newMemStorage() *memStorage { return &memStorage{data: map[string][]byte{}} }

func (s *memStorage) GetWithContext(_ context.Context, key string) ([]byte, error) {
	return s.Get(key)
}

func (s *memStorage) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[key], nil
}

func (s *memStorage) SetWithContext(_ context.Context, key string, val []byte, exp time.Duration) error {
	return s.Set(key, val, exp)
}

func (s *memStorage) Set(key string, val []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append([]byte(nil), val...)
	return nil
}

func (s *memStorage) DeleteWithContext(_ context.Context, key string) error { return s.Delete(key) }

func (s *memStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *memStorage) ResetWithContext(context.Context) error { return s.Reset() }

func (s *memStorage) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = map[string][]byte{}
	return nil
}

func (s *memStorage) Close() error { return nil }

// startedCachePlugin returns the real cache plugin over in-memory storage,
// given a container and started, as the framework would before any module
// starts (monotest does not start plugins itself).
func startedCachePlugin(t *testing.T) *cache.PluginModule {
	t.Helper()
	plugin := cache.NewPluginModuleWithStorage(newMemStorage(), "product:", time.Minute)
	if plugin.Port() != nil {
		t.Fatal("cache port must be nil before the plugin starts")
	}
	plugin.SetContainer(monotest.NewContainer(plugin.Name(), monotest.NewEventBus()))
	if err := plugin.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = plugin.Stop(context.Background()) })
	return plugin
}

func newModule(t *testing.T) *product.Module {
	return product.NewModule(filepath.Join(t.TempDir(), "products.db"))
}

// exerciseCRUD calls every contract the product module provides.
func exerciseCRUD(ctx context.Context, h *monotest.Harness) error {
	c := h.Container()
	created, err := contract.CreateProduct.Call(ctx, c, contract.CreateProductRequest{Name: "Laptop", Price: 999.99, Stock: 5})
	if err != nil {
		return err
	}
	if _, err := contract.ListProducts.Call(ctx, c, contract.ListProductsRequest{Offset: 0, Limit: 10}); err != nil {
		return err
	}
	if _, err := contract.GetProduct.Call(ctx, c, contract.GetProductRequest{ID: created.Product.ID}); err != nil {
		return err
	}
	price := 899.99
	if _, err := contract.UpdateProduct.Call(ctx, c, contract.UpdateProductRequest{ID: created.Product.ID, Price: &price}); err != nil {
		return err
	}
	_, err = contract.DeleteProduct.Call(ctx, c, contract.DeleteProductRequest{ID: created.Product.ID})
	return err
}

// TestProductModuleConformance is the boundary check of the product module,
// with the real cache plugin injected via WithPlugin.
func TestProductModuleConformance(t *testing.T) {
	monotest.CheckModule(t, newModule(t),
		monotest.WithPlugin("cache", startedCachePlugin(t)),
		monotest.WithExercise(exerciseCRUD))
}

// TestProductModuleRequiresCachePlugin checks that Start fails without the plugin.
func TestProductModuleRequiresCachePlugin(t *testing.T) {
	h := monotest.New(t, newModule(t))
	defer func() { _ = h.Stop(context.Background()) }()
	if err := h.Start(context.Background()); err == nil {
		t.Fatal("Start succeeded without the cache plugin")
	}
}

// TestProductModuleRejectsUnstartedPlugin checks that the port is resolved in
// Start, not cached from SetPlugin: an injected but unstarted plugin fails Start.
func TestProductModuleRejectsUnstartedPlugin(t *testing.T) {
	plugin := cache.NewPluginModuleWithStorage(newMemStorage(), "product:", time.Minute)
	plugin.SetContainer(monotest.NewContainer(plugin.Name(), monotest.NewEventBus()))
	h := monotest.New(t, newModule(t), monotest.WithPlugin("cache", plugin))
	defer func() { _ = h.Stop(context.Background()) }()
	if err := h.Start(context.Background()); err == nil {
		t.Fatal("Start succeeded with an unstarted cache plugin")
	}
}

// TestProductContractsCacheAside drives the contracts and checks the cache-hit
// information they return.
func TestProductContractsCacheAside(t *testing.T) {
	ctx := context.Background()
	h := monotest.New(t, newModule(t), monotest.WithPlugin("cache", startedCachePlugin(t)), monotest.Strict())
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()
	c := h.Container()

	created, err := contract.CreateProduct.Call(ctx, c, contract.CreateProductRequest{Name: "Laptop", Price: 999.99, Stock: 5, Category: "Electronics"})
	if err != nil {
		t.Fatal(err)
	}
	id := created.Product.ID
	if id == 0 || created.Product.Name != "Laptop" || created.Product.CreatedAt.IsZero() {
		t.Fatalf("created = %+v", created.Product)
	}

	for i, wantCache := range []bool{false, true} {
		got, err := contract.GetProduct.Call(ctx, c, contract.GetProductRequest{ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if !got.Found || got.FromCache != wantCache || got.Product.Price != 999.99 {
			t.Errorf("get #%d = %+v, want found, from_cache=%v", i+1, got, wantCache)
		}
	}

	for i, wantCache := range []bool{false, true} {
		list, err := contract.ListProducts.Call(ctx, c, contract.ListProductsRequest{Offset: 0, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if list.Total != 1 || len(list.Products) != 1 || list.FromCache != wantCache {
			t.Errorf("list #%d = %+v, want 1 product, from_cache=%v", i+1, list, wantCache)
		}
	}

	price := 899.99
	updated, err := contract.UpdateProduct.Call(ctx, c, contract.UpdateProductRequest{ID: id, Price: &price})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Found || updated.Product.Price != price || updated.Product.Name != "Laptop" {
		t.Errorf("updated = %+v", updated)
	}
	got, err := contract.GetProduct.Call(ctx, c, contract.GetProductRequest{ID: id})
	if err != nil {
		t.Fatal(err)
	}
	if got.FromCache || got.Product.Price != price {
		t.Errorf("get after update = %+v, want a cache miss with the new price", got)
	}

	missing, err := contract.UpdateProduct.Call(ctx, c, contract.UpdateProductRequest{ID: 9999, Price: &price})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Found {
		t.Errorf("update of a missing product reported found")
	}

	if _, err := contract.DeleteProduct.Call(ctx, c, contract.DeleteProductRequest{ID: id}); err != nil {
		t.Fatal(err)
	}
	got, err = contract.GetProduct.Call(ctx, c, contract.GetProductRequest{ID: id})
	if err != nil {
		t.Fatal(err)
	}
	if got.Found {
		t.Errorf("get after delete = %+v, want not found", got)
	}

	if v := h.Report().Violations; len(v) != 0 {
		t.Errorf("boundary violations: %+v", v)
	}
	if n := h.Container().Calls(contract.GetProduct.Name); n != 4 {
		t.Errorf("get-product handled %d calls, want 4", n)
	}
}
