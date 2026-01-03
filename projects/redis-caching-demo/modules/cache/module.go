package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono"
	"github.com/redis/go-redis/v9"
)

// Module provides caching services as a mono module.
type Module struct {
	cache     *Cache
	client    *redis.Client
	redisAddr string
	prefix    string
	ttl       time.Duration
}

// NewModule creates a new cache module with default configuration.
func NewModule(redisAddr string) *Module {
	return &Module{
		redisAddr: redisAddr,
		prefix:    "product:",
		ttl:       5 * time.Minute,
	}
}

// NewModuleWithConfig creates a new cache module with custom configuration.
func NewModuleWithConfig(redisAddr string, prefix string, ttl time.Duration) *Module {
	return &Module{
		redisAddr: redisAddr,
		prefix:    prefix,
		ttl:       ttl,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "cache"
}

// Init initializes the Redis client and creates the cache.
func (m *Module) Init(_ mono.ServiceContainer) error {
	m.client = redis.NewClient(&redis.Options{
		Addr:         m.redisAddr,
		PoolSize:     50,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx := context.Background()
	if err := m.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	m.cache = New(m.client, m.prefix, m.ttl)
	log.Printf("[cache] Connected to Redis at %s (prefix: %s, TTL: %s)", m.redisAddr, m.prefix, m.ttl)

	return nil
}

// Start starts the module (no-op for this module).
func (m *Module) Start(_ context.Context) error {
	log.Println("[cache] Module started")
	return nil
}

// Stop stops the module and closes the Redis connection.
func (m *Module) Stop(_ context.Context) error {
	if m.client != nil {
		if err := m.client.Close(); err != nil {
			log.Printf("[cache] Error closing Redis connection: %v", err)
			return fmt.Errorf("failed to close Redis connection: %w", err)
		}
	}
	log.Println("[cache] Module stopped")
	return nil
}

// GetCache returns the cache instance.
func (m *Module) GetCache() *Cache {
	return m.cache
}

// HealthCheck verifies the Redis connection is healthy.
func (m *Module) HealthCheck(ctx context.Context) error {
	if m.cache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return m.cache.Ping(ctx)
}
