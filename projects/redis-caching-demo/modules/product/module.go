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
	db          *gorm.DB
	repo        *product.Repository
	service     *Service
	cache       *cache.Cache
	dbPath      string
}

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

// SetCache sets the cache instance for the module.
func (m *Module) SetCache(c *cache.Cache) {
	m.cache = c
}

// Init initializes the database and creates the service.
func (m *Module) Init(_ mono.ServiceContainer) error {
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

	// Create service (cache will be set later via SetCache)
	if m.cache != nil {
		m.service = NewService(m.repo, m.cache)
	}

	log.Printf("[product] Database initialized at %s", m.dbPath)
	return nil
}

// Start starts the module.
func (m *Module) Start(_ context.Context) error {
	// Ensure service is created if cache was set after Init
	if m.service == nil && m.cache != nil {
		m.service = NewService(m.repo, m.cache)
	}

	if m.service == nil {
		return fmt.Errorf("product service not initialized: cache not set")
	}

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

// HealthCheck verifies the database connection is healthy.
func (m *Module) HealthCheck(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
