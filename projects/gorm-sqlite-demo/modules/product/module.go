package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProductModule provides product management services via GORM + SQLite.
type ProductModule struct {
	db     *gorm.DB
	repo   *Repository
	dbPath string
}

// Compile-time interface checks.
var _ mono.Module = (*ProductModule)(nil)
var _ mono.ServiceProviderModule = (*ProductModule)(nil)
var _ mono.HealthCheckableModule = (*ProductModule)(nil)

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

// RegisterServices registers request-reply services in the service container.
// The framework automatically prefixes service names with "services.<module>."
// so "create" becomes "services.product.create" in the NATS subject.
func (m *ProductModule) RegisterServices(container mono.ServiceContainer) error {
	if err := helper.RegisterTypedRequestReplyService(
		container, "create", json.Unmarshal, json.Marshal, m.createProduct,
	); err != nil {
		return fmt.Errorf("failed to register create service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "get", json.Unmarshal, json.Marshal, m.getProduct,
	); err != nil {
		return fmt.Errorf("failed to register get service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "list", json.Unmarshal, json.Marshal, m.listProducts,
	); err != nil {
		return fmt.Errorf("failed to register list service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "update", json.Unmarshal, json.Marshal, m.updateProduct,
	); err != nil {
		return fmt.Errorf("failed to register update service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "delete", json.Unmarshal, json.Marshal, m.deleteProduct,
	); err != nil {
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
