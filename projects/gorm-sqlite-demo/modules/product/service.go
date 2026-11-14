package product

import (
	"context"
	"errors"

	"github.com/go-monolith/mono"
	"github.com/google/uuid"

	"github.com/example/gorm-sqlite-demo/modules/product/contract"
)

// Validation errors.
var (
	errNameRequired  = errors.New("name is required")
	errNameEmpty     = errors.New("name cannot be empty")
	errIDRequired    = errors.New("id is required")
	errPriceNegative = errors.New("price must be non-negative")
	errStockNegative = errors.New("stock must be non-negative")
)

// createProduct handles the product.create service request.
func (m *ProductModule) createProduct(ctx context.Context, req contract.CreateProductRequest, _ *mono.Msg) (contract.CreateProductResponse, error) {
	if req.Name == "" {
		return contract.CreateProductResponse{}, errNameRequired
	}
	if req.Price < 0 {
		return contract.CreateProductResponse{}, errPriceNegative
	}
	if req.Stock < 0 {
		return contract.CreateProductResponse{}, errStockNegative
	}

	product := &Product{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}

	if err := m.repo.Create(ctx, product); err != nil {
		return contract.CreateProductResponse{}, err
	}

	return contract.CreateProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		CreatedAt:   product.CreatedAt,
	}, nil
}

// getProduct handles the product.get service request.
func (m *ProductModule) getProduct(ctx context.Context, req contract.GetProductRequest, _ *mono.Msg) (contract.ProductResponse, error) {
	if req.ID == "" {
		return contract.ProductResponse{}, errIDRequired
	}

	product, err := m.repo.FindByID(ctx, req.ID)
	if err != nil {
		return contract.ProductResponse{}, err
	}

	return toProductResponse(product), nil
}

// listProducts handles the product.list service request.
func (m *ProductModule) listProducts(ctx context.Context, _ contract.ListProductsRequest, _ *mono.Msg) (contract.ListProductsResponse, error) {
	products, err := m.repo.FindAll(ctx)
	if err != nil {
		return contract.ListProductsResponse{}, err
	}

	response := contract.ListProductsResponse{
		Products: make([]contract.ProductResponse, 0, len(products)),
		Total:    len(products),
	}

	for _, product := range products {
		response.Products = append(response.Products, toProductResponse(product))
	}

	return response, nil
}

// updateProduct handles the product.update service request.
func (m *ProductModule) updateProduct(ctx context.Context, req contract.UpdateProductRequest, _ *mono.Msg) (contract.ProductResponse, error) {
	if req.ID == "" {
		return contract.ProductResponse{}, errIDRequired
	}

	product, err := m.repo.FindByID(ctx, req.ID)
	if err != nil {
		return contract.ProductResponse{}, err
	}

	if req.Name != nil {
		if *req.Name == "" {
			return contract.ProductResponse{}, errNameEmpty
		}
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		if *req.Price < 0 {
			return contract.ProductResponse{}, errPriceNegative
		}
		product.Price = *req.Price
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			return contract.ProductResponse{}, errStockNegative
		}
		product.Stock = *req.Stock
	}

	if err := m.repo.Update(ctx, product); err != nil {
		return contract.ProductResponse{}, err
	}

	return toProductResponse(product), nil
}

// deleteProduct handles the product.delete service request.
func (m *ProductModule) deleteProduct(ctx context.Context, req contract.DeleteProductRequest, _ *mono.Msg) (contract.DeleteProductResponse, error) {
	if req.ID == "" {
		return contract.DeleteProductResponse{}, errIDRequired
	}

	if err := m.repo.Delete(ctx, req.ID); err != nil {
		return contract.DeleteProductResponse{ID: req.ID}, err
	}

	return contract.DeleteProductResponse{Deleted: true, ID: req.ID}, nil
}

// toProductResponse converts a Product entity to a ProductResponse.
func toProductResponse(product *Product) contract.ProductResponse {
	return contract.ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}
