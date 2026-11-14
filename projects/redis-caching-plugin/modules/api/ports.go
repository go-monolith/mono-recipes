package api

import (
	"context"

	productcontract "github.com/example/redis-caching-demo/modules/product/contract"
)

// ProductPort is the driven port the HTTP handlers need: the product catalog,
// in the product module's contract DTOs. The api module owns it; tests can
// substitute a fake for it. It says nothing about NATS.
type ProductPort interface {
	ListProducts(ctx context.Context, req productcontract.ListProductsRequest) (productcontract.ListProductsResponse, error)
	GetProduct(ctx context.Context, req productcontract.GetProductRequest) (productcontract.GetProductResponse, error)
	CreateProduct(ctx context.Context, req productcontract.CreateProductRequest) (productcontract.CreateProductResponse, error)
	UpdateProduct(ctx context.Context, req productcontract.UpdateProductRequest) (productcontract.UpdateProductResponse, error)
	DeleteProduct(ctx context.Context, req productcontract.DeleteProductRequest) (productcontract.DeleteProductResponse, error)
}
