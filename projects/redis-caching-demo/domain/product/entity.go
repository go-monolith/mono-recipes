// Package product provides the domain entity and repository for products.
package product

import (
	"time"

	"gorm.io/gorm"
)

// Product represents a product in the catalog.
type Product struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"size:1000" json:"description"`
	Price       float64        `gorm:"not null" json:"price"`
	Stock       int            `gorm:"default:0" json:"stock"`
	Category    string         `gorm:"size:100" json:"category"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// CreateProductRequest represents the request to create a product.
type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	Description string  `json:"description" validate:"max=1000"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"gte=0"`
	Category    string  `json:"category" validate:"max=100"`
}

// UpdateProductRequest represents the request to update a product.
type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string  `json:"description,omitempty" validate:"omitempty,max=1000"`
	Price       *float64 `json:"price,omitempty" validate:"omitempty,gt=0"`
	Stock       *int     `json:"stock,omitempty" validate:"omitempty,gte=0"`
	Category    *string  `json:"category,omitempty" validate:"omitempty,max=100"`
}

// ProductResponse represents the response containing a product.
type ProductResponse struct {
	Product   *Product `json:"product"`
	FromCache bool     `json:"from_cache"`
}

// ProductListResponse represents the response containing a list of products.
type ProductListResponse struct {
	Products  []Product `json:"products"`
	Total     int64     `json:"total"`
	FromCache bool      `json:"from_cache"`
}
