// Package contract is the published port of the product module: the DTOs of
// its services and the typed contracts other modules call them through. It is
// the ONLY package of the product module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper. Never its own
// module package and never the GORM-backed domain package, so the DTOs below
// mirror the JSON shape of domain/product.Product instead of embedding it.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// ModuleName is the name of the module that provides these contracts.
const ModuleName = "product"

// Product is the wire representation of a catalog product.
type Product struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListProductsRequest asks for one page of products, ordered by ID.
type ListProductsRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ListProductsResponse is one page of products. FromCache reports whether the
// page was served from Redis (cache hit) or from the database (cache miss).
type ListProductsResponse struct {
	Products  []Product `json:"products"`
	Total     int64     `json:"total"`
	FromCache bool      `json:"from_cache"`
}

// GetProductRequest asks for a product by ID.
type GetProductRequest struct {
	ID uint `json:"id"`
}

// GetProductResponse carries the product, if Found. FromCache reports a cache hit.
type GetProductResponse struct {
	Product   Product `json:"product"`
	Found     bool    `json:"found"`
	FromCache bool    `json:"from_cache"`
}

// CreateProductRequest describes a new product.
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}

// CreateProductResponse carries the created product.
type CreateProductResponse struct {
	Product Product `json:"product"`
}

// UpdateProductRequest is a partial update: nil fields are left unchanged.
type UpdateProductRequest struct {
	ID          uint     `json:"id"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	Stock       *int     `json:"stock,omitempty"`
	Category    *string  `json:"category,omitempty"`
}

// UpdateProductResponse carries the updated product, if Found.
type UpdateProductResponse struct {
	Product Product `json:"product"`
	Found   bool    `json:"found"`
}

// DeleteProductRequest asks to (soft) delete a product by ID.
type DeleteProductRequest struct {
	ID uint `json:"id"`
}

// DeleteProductResponse acknowledges a delete. Deleting a missing product is
// not an error.
type DeleteProductResponse struct {
	ID uint `json:"id"`
}

// Typed contracts, declared once: the product module registers through them,
// consumers call through them. Name, subject (services.product.<name>), kind
// and JSON encoding live here only.
var (
	ListProducts  = helper.RequestReplyService[ListProductsRequest, ListProductsResponse](ModuleName, "list-products")
	GetProduct    = helper.RequestReplyService[GetProductRequest, GetProductResponse](ModuleName, "get-product")
	CreateProduct = helper.RequestReplyService[CreateProductRequest, CreateProductResponse](ModuleName, "create-product")
	UpdateProduct = helper.RequestReplyService[UpdateProductRequest, UpdateProductResponse](ModuleName, "update-product")
	DeleteProduct = helper.RequestReplyService[DeleteProductRequest, DeleteProductResponse](ModuleName, "delete-product")
)
