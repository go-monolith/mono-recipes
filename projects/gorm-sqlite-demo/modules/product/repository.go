package product

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a product is not found.
var ErrNotFound = errors.New("product not found")

// Repository provides access to product storage.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new product repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create saves a new product to the database.
func (r *Repository) Create(product *Product) error {
	if err := r.db.Create(product).Error; err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

// FindByID retrieves a product by its ID.
func (r *Repository) FindByID(id string) (*Product, error) {
	var product Product
	if err := r.db.First(&product, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to find product: %w", err)
	}
	return &product, nil
}

// FindAll retrieves all products.
func (r *Repository) FindAll() ([]*Product, error) {
	var products []*Product
	if err := r.db.Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to find products: %w", err)
	}
	return products, nil
}

// Update updates an existing product.
func (r *Repository) Update(product *Product) error {
	result := r.db.Model(&Product{}).Where("id = ?", product.ID).Updates(product)
	if err := result.Error; err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a product by ID (soft delete).
func (r *Repository) Delete(id string) error {
	result := r.db.Delete(&Product{}, "id = ?", id)
	if err := result.Error; err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
