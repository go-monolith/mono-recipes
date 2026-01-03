package product

import (
	"time"

	"gorm.io/gorm"
)

// Product represents a product in the catalog.
type Product struct {
	ID          string         `gorm:"primarykey;size:36" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Price       float64        `gorm:"not null" json:"price"`
	Stock       int            `gorm:"not null;default:0" json:"stock"`
}

// TableName returns the table name for Product model.
func (Product) TableName() string {
	return "products"
}
