package api

import (
	"context"

	productcontract "github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/go-monolith/mono"
)

// productAdapter is the driven adapter: it implements ProductPort by calling
// the product module's typed contracts with the container received in
// SetDependencyServiceContainer. It is the only place in the api module that
// knows the call crosses a module boundary.
type productAdapter struct{ container mono.ServiceContainer }

var _ ProductPort = productAdapter{}

func (a productAdapter) ListProducts(ctx context.Context, req productcontract.ListProductsRequest) (productcontract.ListProductsResponse, error) {
	return productcontract.ListProducts.Call(ctx, a.container, req)
}

func (a productAdapter) GetProduct(ctx context.Context, req productcontract.GetProductRequest) (productcontract.GetProductResponse, error) {
	return productcontract.GetProduct.Call(ctx, a.container, req)
}

func (a productAdapter) CreateProduct(ctx context.Context, req productcontract.CreateProductRequest) (productcontract.CreateProductResponse, error) {
	return productcontract.CreateProduct.Call(ctx, a.container, req)
}

func (a productAdapter) UpdateProduct(ctx context.Context, req productcontract.UpdateProductRequest) (productcontract.UpdateProductResponse, error) {
	return productcontract.UpdateProduct.Call(ctx, a.container, req)
}

func (a productAdapter) DeleteProduct(ctx context.Context, req productcontract.DeleteProductRequest) (productcontract.DeleteProductResponse, error) {
	return productcontract.DeleteProduct.Call(ctx, a.container, req)
}
