package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"

	"github.com/example/url-shortener-demo/modules/analytics"
	"github.com/example/url-shortener-demo/modules/httpserver"
	"github.com/example/url-shortener-demo/modules/shortener"
)

func main() {
	// Configuration
	httpAddr := getEnv("HTTP_ADDR", ":8080")
	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	jsDir := getEnv("JETSTREAM_DIR", "/tmp/url-shortener-demo")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
		mono.WithJetStreamStorageDir(jsDir),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create kv-jetstream plugin for URL storage
	// The kv-jetstream plugin provides:
	// - Fast key-value storage using embedded NATS JetStream
	// - TTL support for auto-expiring URLs
	// - Revision-based optimistic locking for concurrent updates
	// - No external dependencies - runs embedded in the application
	kvStore, err := kvjetstream.New(kvjetstream.Config{
		Buckets: []kvjetstream.BucketConfig{
			{
				Name:        "urls",
				Description: "Shortened URL mappings",
				// Use memory storage for fast access
				// Switch to FileStorage for persistence across restarts
				Storage: kvjetstream.MemoryStorage,
				// Default TTL for URLs (0 = never expires)
				// Individual URLs can override this
				TTL: 0,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create KV plugin: %v", err)
	}

	// Register the plugin with alias "kv"
	// Modules that implement UsePluginModule will receive this plugin
	// via their SetPlugin(alias, plugin) method before Start() is called
	if err := app.RegisterPlugin(kvStore, "kv"); err != nil {
		log.Fatalf("Failed to register KV plugin: %v", err)
	}

	logger := app.Logger()

	// Create modules
	// Note: Module creation order doesn't matter.
	// The framework automatically resolves dependencies and determines startup/shutdown order.
	shortenerModule := shortener.NewModule(baseURL, logger)
	analyticsModule := analytics.NewModule(logger)
	httpServerModule := httpserver.NewModule(httpAddr, logger)

	// Register modules
	// The framework handles:
	// 1. Plugin injection (kv plugin -> shortener via SetPlugin)
	// 2. Event bus wiring (shortener emits events, analytics consumes)
	// 3. Dependency resolution (httpserver depends on shortener & analytics)
	// 4. Service container injection (via SetDependencyServiceContainer)
	// 5. Lifecycle management (Start/Stop in correct dependency order)
	app.Register(shortenerModule)
	app.Register(analyticsModule)
	app.Register(httpServerModule)

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Printf("URL Shortener Demo started on %s", httpAddr)
	log.Printf("Base URL: %s", baseURL)
	log.Println("Press Ctrl+C to shutdown...")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}

	log.Println("Application stopped successfully")
}

// getEnv returns the environment variable value or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
