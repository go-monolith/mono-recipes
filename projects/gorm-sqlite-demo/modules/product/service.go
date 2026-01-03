package product

import (
	"context"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/google/uuid"
)

// createProduct handles the product.create service request.
func (m *ProductModule) createProduct(_ context.Context, req CreateProductRequest, _ *mono.Msg) (CreateProductResponse, error) {
	// Validate request
	if req.Name == "" {
		return CreateProductResponse{}, fmt.Errorf("name is required")
	}
	if req.Price < 0 {
		return CreateProductResponse{}, fmt.Errorf("price must be non-negative")
	}
	if req.Stock < 0 {
		return CreateProductResponse{}, fmt.Errorf("stock must be non-negative")
	}

	// Create product entity (GORM handles CreatedAt/UpdatedAt automatically)
	product := &Product{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}

	// Save to repository
	if err := m.repo.Create(product); err != nil {
		return CreateProductResponse{}, fmt.Errorf("failed to save product: %w", err)
	}

	return CreateProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		CreatedAt:   product.CreatedAt,
	}, nil
}

// getProduct handles the product.get service request.
func (m *ProductModule) getProduct(_ context.Context, req GetProductRequest, _ *mono.Msg) (ProductResponse, error) {
	if req.ID == "" {
		return ProductResponse{}, fmt.Errorf("id is required")
	}

	product, err := m.repo.FindByID(req.ID)
	if err != nil {
		return ProductResponse{}, err
	}

	return toProductResponse(product), nil
}

// listProducts handles the product.list service request.
func (m *ProductModule) listProducts(_ context.Context, _ ListProductsRequest, _ *mono.Msg) (ListProductsResponse, error) {
	products, err := m.repo.FindAll()
	if err != nil {
		return ListProductsResponse{}, err
	}

	response := ListProductsResponse{
		Products: make([]ProductResponse, 0, len(products)),
		Total:    len(products),
	}

	for _, product := range products {
		response.Products = append(response.Products, toProductResponse(product))
	}

	return response, nil
}

// updateProduct handles the product.update service request.
func (m *ProductModule) updateProduct(_ context.Context, req UpdateProductRequest, _ *mono.Msg) (ProductResponse, error) {
	if req.ID == "" {
		return ProductResponse{}, fmt.Errorf("id is required")
	}

	product, err := m.repo.FindByID(req.ID)
	if err != nil {
		return ProductResponse{}, err
	}

	// Update fields if provided (GORM handles UpdatedAt automatically)
	if req.Name != nil {
		if *req.Name == "" {
			return ProductResponse{}, fmt.Errorf("name cannot be empty")
		}
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		if *req.Price < 0 {
			return ProductResponse{}, fmt.Errorf("price must be non-negative")
		}
		product.Price = *req.Price
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			return ProductResponse{}, fmt.Errorf("stock must be non-negative")
		}
		product.Stock = *req.Stock
	}

	if err := m.repo.Update(product); err != nil {
		return ProductResponse{}, fmt.Errorf("failed to update product: %w", err)
	}

	return toProductResponse(product), nil
}

// deleteProduct handles the product.delete service request.
func (m *ProductModule) deleteProduct(_ context.Context, req DeleteProductRequest, _ *mono.Msg) (DeleteProductResponse, error) {
	if req.ID == "" {
		return DeleteProductResponse{Deleted: false}, fmt.Errorf("id is required")
	}

	if err := m.repo.Delete(req.ID); err != nil {
		return DeleteProductResponse{Deleted: false, ID: req.ID}, err
	}

	return DeleteProductResponse{Deleted: true, ID: req.ID}, nil
}

// toProductResponse converts a Product entity to a ProductResponse.
func toProductResponse(product *Product) ProductResponse {
	return ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}
