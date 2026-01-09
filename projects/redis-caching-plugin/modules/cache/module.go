package cache

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/storage"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/storage/redis/v3"
)

// PluginModule provides caching services as a mono plugin module.
// Plugins start first and stop last, making them ideal for cross-cutting concerns.
type PluginModule struct {
	container types.ServiceContainer
	storage   storage.Storage
	service   CacheService
	redisAddr string
	prefix    string
	ttl       time.Duration
}

// Compile-time interface checks.
var (
	_ mono.PluginModule          = (*PluginModule)(nil)
	_ mono.HealthCheckableModule = (*PluginModule)(nil)
)

// NewPluginModule creates a new cache plugin module with default configuration.
func NewPluginModule(redisAddr string) *PluginModule {
	return NewPluginModuleWithConfig(redisAddr, "product:", 5*time.Minute)
}

// NewPluginModuleWithConfig creates a new cache plugin module with custom configuration.
// The Redis storage is created immediately so Port() can be called before Start().
func NewPluginModuleWithConfig(redisAddr, prefix string, ttl time.Duration) *PluginModule {
	return &PluginModule{
		redisAddr: redisAddr,
		prefix:    prefix,
		ttl:       ttl,
	}
}

// ============================================================
// Module Interface Implementation
// ============================================================

// Name returns the module name.
func (m *PluginModule) Name() string {
	return "cache"
}

// Start is called by the mono framework when starting the plugin.
// Plugins start before regular modules.
func (m *PluginModule) Start(_ context.Context) error {
	// Parse host:port from redisAddr
	host, port := parseRedisAddr(m.redisAddr)
	m.storage = redis.New(redis.Config{
		Host:     host,
		Port:     port,
		PoolSize: 50,
	})
	m.service = NewCacheService(m.storage, m.prefix, m.ttl)
	log.Printf("[cache] Connected to Redis at %s (prefix: %s, TTL: %s)", m.redisAddr, m.prefix, m.ttl)
	log.Println("[cache] Plugin started")
	return nil
}

// Stop stops the plugin and closes the Redis connection.
// Plugins stop after regular modules.
func (m *PluginModule) Stop(_ context.Context) error {
	if m.service != nil {
		if err := m.service.Close(); err != nil {
			log.Printf("[cache] Error closing connection: %v", err)
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	log.Println("[cache] Plugin stopped")
	return nil
}

// ============================================================
// PluginModule Interface Implementation
// ============================================================

// SetContainer sets the service container for this plugin.
func (m *PluginModule) SetContainer(container types.ServiceContainer) {
	m.container = container
}

// Container returns the service container for this plugin.
func (m *PluginModule) Container() types.ServiceContainer {
	return m.container
}

// ============================================================
// Public API (Port Interface)
// ============================================================

// Port returns the CacheService interface for consumers.
// This is the public API that other modules use.
func (m *PluginModule) Port() CacheService {
	return m.service
}

// ============================================================
// Health Check
// ============================================================

// Health returns the current health status.
func (m *PluginModule) Health(ctx context.Context) mono.HealthStatus {
	if m.storage == nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: "storage not initialized",
		}
	}

	// Simple health check: try to get a non-existent key
	_, err := m.storage.Get("__health_check__")
	if err != nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("health check failed: %v", err),
		}
	}

	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"redis_addr": m.redisAddr,
			"prefix":     m.prefix,
			"ttl":        m.ttl.String(),
		},
	}
}

// ============================================================
// Helper Functions
// ============================================================

// parseRedisAddr parses "host:port" into host and port.
// Returns defaults (127.0.0.1:6379) for invalid or missing values.
func parseRedisAddr(addr string) (string, int) {
	const defaultHost = "127.0.0.1"
	const defaultPort = 6379

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return defaultHost, defaultPort
	}

	if host == "" {
		host = defaultHost
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = defaultPort
	}

	return host, port
}
