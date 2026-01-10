// Example: Creating a Custom Plugin Module
//
// This example demonstrates:
// - Implementing the PluginModule interface
// - Creating a plugin configuration
// - Defining a public API (port) for consumers
// - Plugin lifecycle management
// - Using the plugin in a consumer module

package pluginmodule

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
)

// ============================================================
// Custom Cache Plugin Implementation
// ============================================================

// Config holds plugin configuration
type Config struct {
	DefaultTTL time.Duration
	MaxEntries int
}

// PluginModule implements a simple in-memory cache plugin
type CachePlugin struct {
	name      string
	container types.ServiceContainer
	config    Config
	cache     map[string]*cacheEntry
	mu        sync.RWMutex
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// Compile-time interface check
var _ mono.PluginModule = (*CachePlugin)(nil)

// New creates a new cache plugin instance
func NewCachePlugin(config Config) (*CachePlugin, error) {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = time.Hour
	}
	if config.MaxEntries == 0 {
		config.MaxEntries = 10000
	}
	return &CachePlugin{
		name:   "cache",
		config: config,
		cache:  make(map[string]*cacheEntry),
	}, nil
}

// ============================================================
// Module Interface Implementation
// ============================================================

func (p *CachePlugin) Name() string {
	return p.name
}

func (p *CachePlugin) Start(ctx context.Context) error {
	slog.Info("Cache plugin starting",
		"defaultTTL", p.config.DefaultTTL,
		"maxEntries", p.config.MaxEntries)

	// Start background cleanup goroutine
	go p.cleanupLoop(ctx)

	slog.Info("Cache plugin started")
	return nil
}

func (p *CachePlugin) Stop(ctx context.Context) error {
	slog.Info("Cache plugin stopping")

	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear the cache
	p.cache = nil

	slog.Info("Cache plugin stopped")
	return nil
}

// ============================================================
// PluginModule Interface Implementation
// ============================================================

func (p *CachePlugin) SetContainer(container types.ServiceContainer) {
	p.container = container
}

func (p *CachePlugin) Container() types.ServiceContainer {
	return p.container
}

// ============================================================
// Background Cleanup
// ============================================================

func (p *CachePlugin) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.cleanup()
		}
	}
}

func (p *CachePlugin) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	expired := 0
	for key, entry := range p.cache {
		if now.After(entry.expiresAt) {
			delete(p.cache, key)
			expired++
		}
	}

	if expired > 0 {
		slog.Debug("Cache cleanup", "expired", expired, "remaining", len(p.cache))
	}
}

// ============================================================
// Public API (Port Interface)
// ============================================================

// CachePort is the public interface for cache consumers
type CachePort interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
	Size() int
}

// Port returns the public cache interface
func (p *CachePlugin) Port() CachePort {
	return p
}

// Get retrieves a value from the cache
func (p *CachePlugin) Get(key string) ([]byte, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entry, ok := p.cache[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in the cache
func (p *CachePlugin) Set(key string, value []byte, ttl time.Duration) {
	if ttl == 0 {
		ttl = p.config.DefaultTTL
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check max entries limit
	if len(p.cache) >= p.config.MaxEntries {
		// Simple eviction: remove oldest entry
		p.evictOne()
	}

	p.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache
func (p *CachePlugin) Delete(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cache, key)
}

// Size returns the number of entries in the cache
func (p *CachePlugin) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.cache)
}

func (p *CachePlugin) evictOne() {
	// Find and remove the oldest entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range p.cache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(p.cache, oldestKey)
	}
}

// ============================================================
// Consumer Module Using the Plugin
// ============================================================

// UserModule is a module that uses the cache plugin
type UserModule struct {
	cache CachePort
}

// Compile-time interface checks
var (
	_ mono.Module          = (*UserModule)(nil)
	_ mono.UsePluginModule = (*UserModule)(nil)
)

func NewUserModule() *UserModule {
	return &UserModule{}
}

func (m *UserModule) Name() string { return "users" }

// SetPlugin receives the cache plugin
func (m *UserModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "cache" {
		m.cache = plugin.(*CachePlugin).Port()
	}
}

func (m *UserModule) Start(ctx context.Context) error {
	if m.cache == nil {
		return fmt.Errorf("required plugin 'cache' not registered")
	}

	slog.Info("User module started with cache plugin")

	// Use the cache
	m.cache.Set("user:123", []byte(`{"name":"Alice","email":"alice@example.com"}`), 0)
	m.cache.Set("user:456", []byte(`{"name":"Bob","email":"bob@example.com"}`), time.Minute*30)

	return nil
}

func (m *UserModule) Stop(ctx context.Context) error {
	slog.Info("User module stopped")
	return nil
}

// GetUser retrieves a user from cache
func (m *UserModule) GetUser(userID string) ([]byte, bool) {
	return m.cache.Get("user:" + userID)
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Custom Plugin Example ===")
	fmt.Println("Demonstrates: Creating and using a custom plugin")
	fmt.Println()

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create cache plugin
	cachePlugin, err := NewCachePlugin(Config{
		DefaultTTL: time.Hour,
		MaxEntries: 1000,
	})
	if err != nil {
		log.Fatalf("Failed to create cache plugin: %v", err)
	}

	// Register plugin with alias "cache"
	if err := app.RegisterPlugin(cachePlugin, "cache"); err != nil {
		log.Fatalf("Failed to register plugin: %v", err)
	}
	fmt.Println("Cache plugin registered")

	// Create and register consumer module
	userModule := NewUserModule()
	if err := app.Register(userModule); err != nil {
		log.Fatalf("Failed to register module: %v", err)
	}
	fmt.Println("User module registered")

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Println()

	// Test the cache through the module
	fmt.Println("=== Testing Cache ===")

	if data, found := userModule.GetUser("123"); found {
		fmt.Printf("User 123: %s\n", string(data))
	}

	if data, found := userModule.GetUser("456"); found {
		fmt.Printf("User 456: %s\n", string(data))
	}

	if _, found := userModule.GetUser("999"); !found {
		fmt.Println("User 999: not found (expected)")
	}

	fmt.Printf("Cache size: %d entries\n", cachePlugin.Size())

	// Wait for shutdown signal
	fmt.Println("\nPress Ctrl+C to shutdown...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}

	fmt.Println("App stopped successfully")
}
