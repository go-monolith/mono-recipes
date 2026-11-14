package product

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-monolith/mono"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/example/gorm-sqlite-demo/modules/product/contract"
)

// ProductModule provides product management services via GORM + SQLite.
type ProductModule struct {
	db     *gorm.DB
	repo   *Repository
	dbPath string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*ProductModule)(nil)
	_ mono.ServiceProviderModule = (*ProductModule)(nil)
	_ mono.HealthCheckableModule = (*ProductModule)(nil)
)

// NewModule creates a new ProductModule.
func NewModule() *ProductModule {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "products.db"
	}
	return &ProductModule{
		dbPath: dbPath,
	}
}

// Name returns the module name.
func (m *ProductModule) Name() string {
	return "product"
}

// Health performs a health check on the product module.
func (m *ProductModule) Health(ctx context.Context) mono.HealthStatus {
	if m.db == nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: "database not initialized",
		}
	}

	sqlDB, err := m.db.DB()
	if err != nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("failed to get sql.DB: %v", err),
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("database ping failed: %v", err),
		}
	}

	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"driver": "sqlite",
			"path":   m.dbPath,
		},
	}
}

// RegisterServices is the module's driving adapter: it binds each handler to
// the module's own typed contract (see contract/contract.go), so service names,
// subjects (services.product.<service>) and JSON encoding are declared once.
func (m *ProductModule) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.CreateProduct.Register(container, m.createProduct); err != nil {
		return fmt.Errorf("failed to register create service: %w", err)
	}
	if err := contract.GetProduct.Register(container, m.getProduct); err != nil {
		return fmt.Errorf("failed to register get service: %w", err)
	}
	if err := contract.ListProducts.Register(container, m.listProducts); err != nil {
		return fmt.Errorf("failed to register list service: %w", err)
	}
	if err := contract.UpdateProduct.Register(container, m.updateProduct); err != nil {
		return fmt.Errorf("failed to register update service: %w", err)
	}
	if err := contract.DeleteProduct.Register(container, m.deleteProduct); err != nil {
		return fmt.Errorf("failed to register delete service: %w", err)
	}

	log.Printf("[product] Registered services: services.product.{create,get,list,update,delete}")
	return nil
}

// Start initializes the database connection and runs migrations.
func (m *ProductModule) Start(_ context.Context) error {
	log.Printf("[product] Connecting to SQLite database: %s", m.dbPath)

	// Configure GORM logger based on environment
	logLevel := logger.Silent
	if os.Getenv("DB_DEBUG") == "true" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(sqlite.Open(m.dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Request-reply handlers run concurrently on the framework's worker pool
	// (mono v0.1.1+). SQLite allows a single writer at a time, so funnel all
	// access through one connection: concurrent requests queue in
	// database/sql instead of contending for SQLite's file lock.
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	m.db = db

	// Run auto-migrations
	if err := m.db.AutoMigrate(&Product{}); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize repository
	m.repo = NewRepository(m.db)

	log.Println("[product] Module started successfully")
	return nil
}

// Stop gracefully closes the database connection.
func (m *ProductModule) Stop(_ context.Context) error {
	if m.db == nil {
		return nil
	}

	log.Println("[product] Closing database connection...")

	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	log.Println("[product] Database connection closed")
	return nil
}
