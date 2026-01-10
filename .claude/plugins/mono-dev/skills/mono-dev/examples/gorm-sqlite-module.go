// gorm-sqlite-module.go demonstrates GORM with SQLite integration
package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Product entity
type Product struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"size:1000" json:"description"`
	Price       float64        `gorm:"not null" json:"price"`
	Stock       int            `gorm:"default:0" json:"stock"`
	Active      bool           `gorm:"default:true" json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Module provides product management via GORM + SQLite
type ProductModule struct {
	db     *gorm.DB
	dbPath string
}

// Compile-time interface checks
var (
	_ mono.Module                = (*ProductModule)(nil)
	_ mono.ServiceProviderModule = (*ProductModule)(nil)
	_ mono.HealthCheckableModule = (*ProductModule)(nil)
)

// NewModule creates a new ProductModule
func NewModule() *ProductModule {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "products.db"
	}
	return &ProductModule{
		dbPath: dbPath,
	}
}

// Name returns the module name
func (m *ProductModule) Name() string { return "product" }

// Health performs a health check
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

// RegisterServices registers CRUD services
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

// Start initializes the database connection
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

	// Auto-migrate models
	if err := m.db.AutoMigrate(&Product{}); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("[product] Module started successfully")
	return nil
}

// Stop closes the database connection
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

// Request/Response types
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type GetProductRequest struct {
	ID uint `json:"id"`
}

type ListProductsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type UpdateProductRequest struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price,omitempty"`
	Stock       int     `json:"stock,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

type DeleteProductRequest struct {
	ID uint `json:"id"`
}

type ProductResponse struct {
	Product *Product `json:"product,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type ProductListResponse struct {
	Products []Product `json:"products"`
	Total    int64     `json:"total"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Service handlers
func (m *ProductModule) createProduct(ctx context.Context, req CreateProductRequest, _ *mono.Msg) (ProductResponse, error) {
	product := Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Active:      true,
	}

	if err := m.db.Create(&product).Error; err != nil {
		return ProductResponse{Error: err.Error()}, nil
	}

	return ProductResponse{Product: &product}, nil
}

func (m *ProductModule) getProduct(ctx context.Context, req GetProductRequest, _ *mono.Msg) (ProductResponse, error) {
	var product Product
	if err := m.db.First(&product, req.ID).Error; err != nil {
		return ProductResponse{Error: "product not found"}, nil
	}

	return ProductResponse{Product: &product}, nil
}

func (m *ProductModule) listProducts(ctx context.Context, req ListProductsRequest, _ *mono.Msg) (ProductListResponse, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var products []Product
	var total int64

	m.db.Model(&Product{}).Count(&total)
	m.db.Limit(limit).Offset(req.Offset).Find(&products)

	return ProductListResponse{Products: products, Total: total}, nil
}

func (m *ProductModule) updateProduct(ctx context.Context, req UpdateProductRequest, _ *mono.Msg) (ProductResponse, error) {
	var product Product
	if err := m.db.First(&product, req.ID).Error; err != nil {
		return ProductResponse{Error: "product not found"}, nil
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}
	if req.Active != nil {
		product.Active = *req.Active
	}

	if err := m.db.Save(&product).Error; err != nil {
		return ProductResponse{Error: err.Error()}, nil
	}

	return ProductResponse{Product: &product}, nil
}

func (m *ProductModule) deleteProduct(ctx context.Context, req DeleteProductRequest, _ *mono.Msg) (DeleteResponse, error) {
	if err := m.db.Delete(&Product{}, req.ID).Error; err != nil {
		return DeleteResponse{Success: false, Error: err.Error()}, nil
	}

	return DeleteResponse{Success: true}, nil
}
