// Package contract is the published port of the product module: the DTOs of
// its request-reply services and the typed contracts that both the module
// (provider side) and any caller (consumer side) use. It is the ONLY package
// of the product module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper. Never the product
// module package itself (GORM entities stay private to the module).
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// Typed request-reply contracts. Each one is declared once: the product module
// registers its handler through it and callers call through it, so the
// service name, the services.product.<service> subject and the JSON encoding
// live here only.
var (
	// CreateProduct is services.product.create.
	CreateProduct = helper.RequestReplyService[CreateProductRequest, CreateProductResponse]("product", "create")
	// GetProduct is services.product.get.
	GetProduct = helper.RequestReplyService[GetProductRequest, ProductResponse]("product", "get")
	// ListProducts is services.product.list.
	ListProducts = helper.RequestReplyService[ListProductsRequest, ListProductsResponse]("product", "list")
	// UpdateProduct is services.product.update.
	UpdateProduct = helper.RequestReplyService[UpdateProductRequest, ProductResponse]("product", "update")
	// DeleteProduct is services.product.delete.
	DeleteProduct = helper.RequestReplyService[DeleteProductRequest, DeleteProductResponse]("product", "delete")
)

// CreateProductRequest is the request for creating a product.
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// CreateProductResponse is the response after creating a product.
type CreateProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetProductRequest is the request for getting a product.
type GetProductRequest struct {
	ID string `json:"id"`
}

// ProductResponse represents a product in responses.
type ProductResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListProductsRequest is the request for listing products.
type ListProductsRequest struct{}

// ListProductsResponse is the response containing a list of products.
type ListProductsResponse struct {
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
}

// UpdateProductRequest is the request for updating a product.
type UpdateProductRequest struct {
	ID          string   `json:"id"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	Stock       *int     `json:"stock,omitempty"`
}

// DeleteProductRequest is the request for deleting a product.
type DeleteProductRequest struct {
	ID string `json:"id"`
}

// DeleteProductResponse is the response after deleting a product.
type DeleteProductResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}
