package ratelimit

import (
	"context"
	"fmt"
	"log"

	"github.com/example/rate-limiting-demo/domain/ratelimit"
	"github.com/go-monolith/mono"
	"github.com/redis/go-redis/v9"
)

// Module provides rate limiting services as a mono module.
type Module struct {
	client     *redis.Client
	middleware *Middleware
	config     ratelimit.MiddlewareConfig
	redisAddr  string
}

// NewModule creates a new rate limiting module.
func NewModule(redisAddr string) *Module {
	return &Module{
		redisAddr: redisAddr,
		config:    ratelimit.DefaultMiddlewareConfig(),
	}
}

// NewModuleWithConfig creates a new rate limiting module with custom configuration.
func NewModuleWithConfig(redisAddr string, config ratelimit.MiddlewareConfig) *Module {
	return &Module{
		redisAddr: redisAddr,
		config:    config,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "rate-limiter"
}

// Init initializes the Redis client and creates the middleware.
func (m *Module) Init(_ mono.ServiceContainer) error {
	m.client = redis.NewClient(&redis.Options{
		Addr: m.redisAddr,
	})

	// Test connection
	ctx := context.Background()
	if err := m.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	m.middleware = NewMiddleware(m.client, m.config)
	log.Printf("[rate-limiter] Connected to Redis at %s", m.redisAddr)

	return nil
}

// Start starts the module (no-op for this module).
func (m *Module) Start(_ context.Context) error {
	log.Println("[rate-limiter] Module started")
	return nil
}

// Stop stops the module and closes the Redis connection.
func (m *Module) Stop(_ context.Context) error {
	if m.client != nil {
		if err := m.client.Close(); err != nil {
			log.Printf("[rate-limiter] Error closing Redis connection: %v", err)
		}
	}
	log.Println("[rate-limiter] Module stopped")
	return nil
}

// GetMiddleware returns the rate limiting middleware.
func (m *Module) GetMiddleware() *Middleware {
	return m.middleware
}

// GetClient returns the Redis client (for testing or advanced use).
func (m *Module) GetClient() *redis.Client {
	return m.client
}

// HealthCheck verifies the Redis connection is healthy.
func (m *Module) HealthCheck(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	return m.client.Ping(ctx).Err()
}
