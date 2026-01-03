package product

import "time"

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
