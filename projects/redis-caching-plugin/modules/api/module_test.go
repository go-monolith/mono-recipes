package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/redis-caching-demo/modules/api"
	productcontract "github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"
)

// client opens a fresh connection per request and closes it afterwards. With
// pooled keep-alive connections the transport may dial a spare connection that
// never sends a request; fasthttp's shutdown waits up to 5s for such a "new"
// connection, which would make Stop timings flaky.
var client = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 30 * time.Second}

// standInProduct is a NATS-free stand-in for the product module. Every
// service the api module consumes is registered through the product module's
// own contract, backed by a tiny in-memory catalog. Product 1 exists; GET
// reports a cache hit on every call after the first, like the real module.
func standInProduct(t *testing.T) *monotest.Container {
	t.Helper()
	c := monotest.NewContainer(productcontract.ModuleName, monotest.NewEventBus())
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	laptop := productcontract.Product{ID: 1, Name: "Laptop", Price: 999.99, Stock: 5, Category: "Electronics", CreatedAt: now, UpdatedAt: now}
	gets := 0

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(productcontract.ListProducts.Register(c, func(_ context.Context, req productcontract.ListProductsRequest, _ *types.Msg) (productcontract.ListProductsResponse, error) {
		return productcontract.ListProductsResponse{Products: []productcontract.Product{laptop}, Total: 1, FromCache: req.Offset > 0}, nil
	}))
	must(productcontract.GetProduct.Register(c, func(_ context.Context, req productcontract.GetProductRequest, _ *types.Msg) (productcontract.GetProductResponse, error) {
		if req.ID != laptop.ID {
			return productcontract.GetProductResponse{}, nil
		}
		gets++
		return productcontract.GetProductResponse{Product: laptop, Found: true, FromCache: gets > 1}, nil
	}))
	must(productcontract.CreateProduct.Register(c, func(_ context.Context, req productcontract.CreateProductRequest, _ *types.Msg) (productcontract.CreateProductResponse, error) {
		return productcontract.CreateProductResponse{Product: productcontract.Product{ID: 2, Name: req.Name, Price: req.Price}}, nil
	}))
	must(productcontract.UpdateProduct.Register(c, func(_ context.Context, req productcontract.UpdateProductRequest, _ *types.Msg) (productcontract.UpdateProductResponse, error) {
		if req.ID != laptop.ID {
			return productcontract.UpdateProductResponse{}, nil
		}
		p := laptop
		if req.Price != nil {
			p.Price = *req.Price
		}
		return productcontract.UpdateProductResponse{Product: p, Found: true}, nil
	}))
	must(productcontract.DeleteProduct.Register(c, func(_ context.Context, req productcontract.DeleteProductRequest, _ *types.Msg) (productcontract.DeleteProductResponse, error) {
		return productcontract.DeleteProductResponse{ID: req.ID}, nil
	}))
	return c
}

// do sends one HTTP request to the running api module and decodes the JSON body.
func do(ctx context.Context, m *api.Module, method, path, body string) (int, map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://"+m.Addr()+path, strings.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("%s %s: decode %q: %w", method, path, raw, err)
	}
	return resp.StatusCode, out, nil
}

// exerciseAllRoutes drives every HTTP route, so every consumed product
// contract is called through the adapter at least once.
func exerciseAllRoutes(m *api.Module) func(ctx context.Context, h *monotest.Harness) error {
	return func(ctx context.Context, _ *monotest.Harness) error {
		cases := []struct {
			method, path, body string
			want               int
		}{
			{http.MethodGet, "/health", "", http.StatusOK},
			{http.MethodGet, "/api/v1/products", "", http.StatusOK},
			{http.MethodGet, "/api/v1/products/1", "", http.StatusOK},
			{http.MethodPost, "/api/v1/products", `{"name":"Mouse","price":29.99}`, http.StatusCreated},
			{http.MethodPut, "/api/v1/products/1", `{"price":899.99}`, http.StatusOK},
			{http.MethodDelete, "/api/v1/products/1", "", http.StatusOK},
		}
		for _, tc := range cases {
			status, _, err := do(ctx, m, tc.method, tc.path, tc.body)
			if err != nil {
				return err
			}
			if status != tc.want {
				return fmt.Errorf("%s %s: status %d, want %d", tc.method, tc.path, status, tc.want)
			}
		}
		return nil
	}
}

// TestAPIModuleConformance is the boundary check of the api module: it must
// declare product, consume only contracts product provides, call each of them
// through its read-only dependency container, and start and stop cleanly.
func TestAPIModuleConformance(t *testing.T) {
	m := api.NewModule(0) // port 0: any free port
	monotest.CheckModule(t, m,
		monotest.WithDependency(productcontract.ModuleName, standInProduct(t)),
		monotest.WithExercise(exerciseAllRoutes(m)))
}

// TestAPIResponses checks that the handlers keep the HTTP response shape,
// including the cache-hit information returned by the product module.
func TestAPIResponses(t *testing.T) {
	ctx := context.Background()
	product := standInProduct(t)
	m := api.NewModule(0)
	h := monotest.New(t, m, monotest.WithDependency(productcontract.ModuleName, product), monotest.Strict())
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	// First GET is a miss, second a hit.
	for i, wantCache := range []bool{false, true} {
		status, body, err := do(ctx, m, http.MethodGet, "/api/v1/products/1", "")
		if err != nil {
			t.Fatal(err)
		}
		if status != http.StatusOK {
			t.Fatalf("get #%d: status %d", i+1, status)
		}
		if body["from_cache"] != wantCache {
			t.Errorf("get #%d: from_cache = %v, want %v", i+1, body["from_cache"], wantCache)
		}
		if _, ok := body["duration_ms"]; !ok {
			t.Errorf("get #%d: duration_ms missing", i+1)
		}
		prod, _ := body["product"].(map[string]any)
		if prod["name"] != "Laptop" || prod["created_at"] == nil {
			t.Errorf("get #%d: product = %v", i+1, body["product"])
		}
	}

	status, body, err := do(ctx, m, http.MethodGet, "/api/v1/products?offset=0&limit=500", "")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || body["limit"] != float64(100) || body["total"] != float64(1) || body["from_cache"] != false {
		t.Errorf("list: status %d body %v", status, body)
	}

	status, _, err = do(ctx, m, http.MethodGet, "/api/v1/products/42", "")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusNotFound {
		t.Errorf("missing product: status %d, want 404", status)
	}

	status, _, err = do(ctx, m, http.MethodPut, "/api/v1/products/42", `{"price":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusNotFound {
		t.Errorf("update missing product: status %d, want 404", status)
	}

	status, body, err = do(ctx, m, http.MethodPost, "/api/v1/products", `{"name":"","price":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusBadRequest || body["message"] != "Name is required" {
		t.Errorf("create without name: status %d body %v", status, body)
	}

	if got := product.Calls(productcontract.GetProduct.Name); got != 3 {
		t.Errorf("get-product called %d times, want 3", got)
	}
	if v := h.Report().Violations; len(v) != 0 {
		t.Errorf("boundary violations: %+v", v)
	}
}
