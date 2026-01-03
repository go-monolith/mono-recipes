package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	apimod "github.com/example/redis-caching-demo/modules/api"
	cachemod "github.com/example/redis-caching-demo/modules/cache"
	productmod "github.com/example/redis-caching-demo/modules/product"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	// Load configuration from environment
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	dbPath := getEnv("DB_PATH", "./products.db")
	httpPort := getEnvInt("HTTP_PORT", 3000)
	cacheTTL := getEnvDuration("CACHE_TTL", 5*time.Minute)
	cachePrefix := getEnv("CACHE_PREFIX", "product:")

	log.Println("=== Redis Caching Demo ===")
	log.Printf("Redis: %s", redisAddr)
	log.Printf("Database: %s", dbPath)
	log.Printf("HTTP Port: %d", httpPort)
	log.Printf("Cache TTL: %s", cacheTTL)
	log.Printf("Cache Prefix: %s", cachePrefix)

	// Create modules
	cacheModule := cachemod.NewModuleWithConfig(redisAddr, cachePrefix, cacheTTL)
	productModule := productmod.NewModule(dbPath)
	apiModule := apimod.NewModule(httpPort)

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
	)
	if err != nil {
		log.Fatalf("Failed to create mono application: %v", err)
	}

	// Register modules
	app.Register(cacheModule)
	app.Register(productModule)
	app.Register(apiModule)

	// Start modules (this handles Init and Start)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	// Wire up dependencies after start
	// Cache module must be wired to product module
	productModule.SetCache(cacheModule.GetCache())

	// Product module must be wired to API module
	apiModule.SetProductModule(productModule)

	log.Println("=== Application Started ===")
	log.Printf("API available at http://localhost:%d", httpPort)
	log.Println("Endpoints:")
	log.Println("  GET    /health              - Health check")
	log.Println("  GET    /api/v1/products     - List products (cached)")
	log.Println("  GET    /api/v1/products/:id - Get product (cached)")
	log.Println("  POST   /api/v1/products     - Create product")
	log.Println("  PUT    /api/v1/products/:id - Update product")
	log.Println("  DELETE /api/v1/products/:id - Delete product")
	log.Println("  GET    /api/v1/cache/stats  - Cache statistics")
	log.Println("  POST   /api/v1/cache/stats/reset - Reset cache stats")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown")

	// Setup graceful shutdown using gelmium/graceful-shutdown
	wait := gfshutdown.GracefulShutdown(
		context.Background(),
		shutdownTimeout,
		map[string]gfshutdown.Operation{
			"mono-app": func(ctx context.Context) error {
				log.Println("Graceful shutdown initiated...")
				return app.Stop(ctx)
			},
		},
	)

	// Wait for shutdown signal and exit with appropriate code
	exitCode := <-wait
	log.Printf("Application exited with code: %d", exitCode)
	os.Exit(exitCode)
}

// getEnv returns environment variable value or default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid int value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvDuration returns environment variable as duration or default.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Warning: invalid duration value for %s: %s, using default: %s", key, value, defaultValue)
	}
	return defaultValue
}
