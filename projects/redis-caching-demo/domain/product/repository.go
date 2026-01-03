package product

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Repository provides database operations for products.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new product repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new product in the database.
func (r *Repository) Create(ctx context.Context, product *Product) error {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

// GetByID retrieves a product by its ID.
func (r *Repository) GetByID(ctx context.Context, id uint) (*Product, error) {
	var product Product
	if err := r.db.WithContext(ctx).First(&product, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	return &product, nil
}

// List retrieves all products with optional pagination.
// Results are ordered by ID for consistent pagination.
func (r *Repository) List(ctx context.Context, offset, limit int) ([]Product, int64, error) {
	var products []Product
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).Model(&Product{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Get products with pagination, ordered by ID for consistent results
	query := r.db.WithContext(ctx).Order("id ASC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}

	return products, total, nil
}

// Update updates an existing product.
func (r *Repository) Update(ctx context.Context, product *Product) error {
	if err := r.db.WithContext(ctx).Save(product).Error; err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	return nil
}

// Delete soft-deletes a product by its ID.
func (r *Repository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Product{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil // Product not found, consider it deleted
	}
	return nil
}

// Migrate runs database migrations for the product table.
func (r *Repository) Migrate() error {
	return r.db.AutoMigrate(&Product{})
}
