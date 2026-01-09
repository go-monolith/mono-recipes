package product

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	product := &Product{
		ID:          uuid.New().String(),
		Name:        "Test Product",
		Description: "A test product",
		Price:       19.99,
		Stock:       100,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(product)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify product was created
	var found Product
	if err := db.First(&found, "id = ?", product.ID).Error; err != nil {
		t.Fatalf("failed to find created product: %v", err)
	}

	if found.Name != product.Name {
		t.Errorf("expected name %q, got %q", product.Name, found.Name)
	}
	if found.Price != product.Price {
		t.Errorf("expected price %v, got %v", product.Price, found.Price)
	}
}

func TestRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// Create a test product
	product := &Product{
		ID:          uuid.New().String(),
		Name:        "FindByID Test",
		Description: "Test description",
		Price:       29.99,
		Stock:       50,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	t.Run("existing product", func(t *testing.T) {
		found, err := repo.FindByID(product.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.ID != product.ID {
			t.Errorf("expected ID %q, got %q", product.ID, found.ID)
		}
		if found.Name != product.Name {
			t.Errorf("expected name %q, got %q", product.Name, found.Name)
		}
	})

	t.Run("non-existent product", func(t *testing.T) {
		_, err := repo.FindByID("non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent product, got nil")
		}
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	t.Run("empty database", func(t *testing.T) {
		products, err := repo.FindAll()
		if err != nil {
			t.Fatalf("FindAll() error = %v", err)
		}
		if len(products) != 0 {
			t.Errorf("expected 0 products, got %d", len(products))
		}
	})

	// Create test products
	for i := 0; i < 3; i++ {
		product := &Product{
			ID:        uuid.New().String(),
			Name:      "Product " + string(rune('A'+i)),
			Price:     float64(10 + i),
			Stock:     i * 10,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(product).Error; err != nil {
			t.Fatalf("failed to create test product: %v", err)
		}
	}

	t.Run("with products", func(t *testing.T) {
		products, err := repo.FindAll()
		if err != nil {
			t.Fatalf("FindAll() error = %v", err)
		}
		if len(products) != 3 {
			t.Errorf("expected 3 products, got %d", len(products))
		}
	})
}

func TestRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// Create a test product
	product := &Product{
		ID:          uuid.New().String(),
		Name:        "Original Name",
		Description: "Original description",
		Price:       19.99,
		Stock:       100,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	t.Run("update existing product", func(t *testing.T) {
		product.Name = "Updated Name"
		product.Price = 29.99

		err := repo.Update(product)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		// Verify update
		var found Product
		if err := db.First(&found, "id = ?", product.ID).Error; err != nil {
			t.Fatalf("failed to find updated product: %v", err)
		}

		if found.Name != "Updated Name" {
			t.Errorf("expected name %q, got %q", "Updated Name", found.Name)
		}
		if found.Price != 29.99 {
			t.Errorf("expected price %v, got %v", 29.99, found.Price)
		}
	})

	t.Run("update non-existent product", func(t *testing.T) {
		nonExistent := &Product{
			ID:    "non-existent-id",
			Name:  "Should Not Work",
			Price: 99.99,
		}
		err := repo.Update(nonExistent)
		if err == nil {
			t.Error("expected error for non-existent product, got nil")
		}
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// Create a test product
	product := &Product{
		ID:        uuid.New().String(),
		Name:      "To Be Deleted",
		Price:     9.99,
		Stock:     10,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}

	t.Run("delete existing product", func(t *testing.T) {
		err := repo.Delete(product.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify soft delete (product exists with deleted_at set)
		var found Product
		err = db.Unscoped().First(&found, "id = ?", product.ID).Error
		if err != nil {
			t.Fatalf("failed to find deleted product: %v", err)
		}
		if !found.DeletedAt.Valid {
			t.Error("expected DeletedAt to be set after soft delete")
		}

		// Verify product is not returned by normal query
		_, err = repo.FindByID(product.ID)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent product", func(t *testing.T) {
		err := repo.Delete("non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent product, got nil")
		}
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
