package product

import (
	"context"
	"fmt"
	"log"

	"github.com/example/redis-caching-demo/domain/product"
	"github.com/example/redis-caching-demo/modules/cache"
	"github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/go-monolith/mono"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// cachePortProvider is what the product module needs from the "cache" plugin:
// something that hands out the CacheService port. *cache.PluginModule
// implements it.
type cachePortProvider interface {
	Port() cache.CacheService
}

// Module provides product services as a mono module. Other modules reach it
// only through the typed contracts in modules/product/contract.
type Module struct {
	db          *gorm.DB
	repo        *product.Repository
	service     *Service
	cachePlugin cachePortProvider
	dbPath      string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
	_ mono.HealthCheckableModule = (*Module)(nil)
	_ mono.UsePluginModule       = (*Module)(nil)
)

// NewModule creates a new product module.
func NewModule(dbPath string) *Module {
	return &Module{
		dbPath: dbPath,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return contract.ModuleName
}

// SetPlugin receives plugin instances from the mono framework.
// It is called before RegisterServices and Start. Only the plugin reference is
// stored here: the plugin's port is resolved in Start, so the module never
// depends on whether (or in which order) the plugin itself was started.
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias != "cache" {
		return
	}
	if p, ok := plugin.(cachePortProvider); ok {
		m.cachePlugin = p
		log.Println("[product] Cache plugin injected")
	}
}

// RegisterServices is the driving adapter: it binds the typed handlers in
// handlers.go to the module's own contracts. The handlers use m.service, which
// is created in Start; the framework dispatches services only after every
// module has started.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.ListProducts.Register(container, m.listProducts); err != nil {
		return err
	}
	if err := contract.GetProduct.Register(container, m.getProduct); err != nil {
		return err
	}
	if err := contract.CreateProduct.Register(container, m.createProduct); err != nil {
		return err
	}
	if err := contract.UpdateProduct.Register(container, m.updateProduct); err != nil {
		return err
	}
	if err := contract.DeleteProduct.Register(container, m.deleteProduct); err != nil {
		return err
	}
	log.Println("[product] Registered services: list-products, get-product, create-product, update-product, delete-product")
	return nil
}

// Start initializes the database and creates the service.
func (m *Module) Start(_ context.Context) error {
	// Resolve the cache port now that the plugin has started.
	if m.cachePlugin == nil {
		return fmt.Errorf("cache plugin not set - ensure 'cache' plugin is registered")
	}
	cacheService := m.cachePlugin.Port()
	if cacheService == nil {
		return fmt.Errorf("cache plugin is not started - its port is not available")
	}

	// Open SQLite database
	db, err := gorm.Open(sqlite.Open(m.dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	m.db = db
	m.repo = product.NewRepository(db)

	// Run migrations
	if err := m.repo.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create service with cache
	m.service = NewService(m.repo, cacheService)

	log.Printf("[product] Database initialized at %s", m.dbPath)
	log.Println("[product] Module started")
	return nil
}

// Stop stops the module and closes the database connection.
func (m *Module) Stop(_ context.Context) error {
	if m.db != nil {
		sqlDB, err := m.db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
	log.Println("[product] Module stopped")
	return nil
}

// Health returns the current health status.
func (m *Module) Health(ctx context.Context) mono.HealthStatus {
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
			Message: fmt.Sprintf("database error: %v", err),
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
			"db_path": m.dbPath,
		},
	}
}
