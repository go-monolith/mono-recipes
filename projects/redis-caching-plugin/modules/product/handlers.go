package product

import (
	"context"

	"github.com/example/redis-caching-demo/domain/product"
	"github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/go-monolith/mono"
)

// Typed request-reply handlers registered in RegisterServices. They translate
// between the published contract DTOs and the domain types, and delegate to the
// cache-aside Service.

func (m *Module) listProducts(ctx context.Context, req contract.ListProductsRequest, _ *mono.Msg) (contract.ListProductsResponse, error) {
	products, total, fromCache, err := m.service.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return contract.ListProductsResponse{}, err
	}
	out := make([]contract.Product, len(products))
	for i := range products {
		out[i] = toContract(&products[i])
	}
	return contract.ListProductsResponse{Products: out, Total: total, FromCache: fromCache}, nil
}

func (m *Module) getProduct(ctx context.Context, req contract.GetProductRequest, _ *mono.Msg) (contract.GetProductResponse, error) {
	p, fromCache, err := m.service.GetByID(ctx, req.ID)
	if err != nil {
		return contract.GetProductResponse{}, err
	}
	if p == nil {
		return contract.GetProductResponse{Found: false}, nil
	}
	return contract.GetProductResponse{Product: toContract(p), Found: true, FromCache: fromCache}, nil
}

func (m *Module) createProduct(ctx context.Context, req contract.CreateProductRequest, _ *mono.Msg) (contract.CreateProductResponse, error) {
	p, err := m.service.Create(ctx, &product.CreateProductRequest{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
	})
	if err != nil {
		return contract.CreateProductResponse{}, err
	}
	return contract.CreateProductResponse{Product: toContract(p)}, nil
}

func (m *Module) updateProduct(ctx context.Context, req contract.UpdateProductRequest, _ *mono.Msg) (contract.UpdateProductResponse, error) {
	p, err := m.service.Update(ctx, req.ID, &product.UpdateProductRequest{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
	})
	if err != nil {
		return contract.UpdateProductResponse{}, err
	}
	if p == nil {
		return contract.UpdateProductResponse{Found: false}, nil
	}
	return contract.UpdateProductResponse{Product: toContract(p), Found: true}, nil
}

func (m *Module) deleteProduct(ctx context.Context, req contract.DeleteProductRequest, _ *mono.Msg) (contract.DeleteProductResponse, error) {
	if err := m.service.Delete(ctx, req.ID); err != nil {
		return contract.DeleteProductResponse{}, err
	}
	return contract.DeleteProductResponse{ID: req.ID}, nil
}

// toContract maps the GORM entity to its wire representation.
func toContract(p *product.Product) contract.Product {
	return contract.Product{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		Category:    p.Category,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
