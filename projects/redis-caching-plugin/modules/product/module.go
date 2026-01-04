package product

import (
	"context"
	"fmt"
	"log"

	"github.com/example/redis-caching-demo/domain/product"
	"github.com/example/redis-caching-demo/modules/cache"
	"github.com/go-monolith/mono"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Module provides product services as a mono module.
type Module struct {
	db      *gorm.DB
	repo    *product.Repository
	service *Service
	cache   cache.CacheService
	dbPath  string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*Module)(nil)
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
	return "product"
}

// SetPlugin receives plugin instances from the mono framework.
// This is called before Start() to inject plugin dependencies.
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "cache" {
		// Type assert to cache.PluginModule to access Port()
		if cachePlugin, ok := plugin.(*cache.PluginModule); ok {
			m.cache = cachePlugin.Port()
			log.Println("[product] Cache plugin injected")
		}
	}
}

// Start initializes the database and creates the service.
func (m *Module) Start(_ context.Context) error {
	// Verify cache plugin was injected
	if m.cache == nil {
		return fmt.Errorf("cache plugin not set - ensure 'cache' plugin is registered")
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
	m.service = NewService(m.repo, m.cache)

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

// GetService returns the product service.
func (m *Module) GetService() *Service {
	return m.service
}

// GetRepository returns the product repository.
func (m *Module) GetRepository() *product.Repository {
	return m.repo
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
