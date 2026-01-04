package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/rate-limiting-middleware/middleware/ratelimit"
	"github.com/example/rate-limiting-middleware/modules/api"
)

func main() {
	// Configuration from environment
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	logger := app.Logger()

	// Create rate limiting middleware with service-specific limits
	rateLimitMiddleware, err := ratelimit.New(
		ratelimit.WithRedisAddr(redisAddr),
		ratelimit.WithRedisPassword(redisPassword),
		// Default: 100 requests per minute
		ratelimit.WithDefaultLimit(100, time.Minute),
		// api.getData: 100 requests per minute (uses default)
		// api.createOrder: 50 requests per minute (more restrictive)
		ratelimit.WithServiceLimit(api.ServiceCreateOrder, 50, time.Minute),
		// api.getStatus: 200 requests per minute (less restrictive for health checks)
		ratelimit.WithServiceLimit(api.ServiceGetStatus, 200, time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create rate limiting middleware: %v", err)
	}

	// Create API module
	apiModule := api.NewModule(logger)

	// Register middleware BEFORE regular modules
	// Middleware must be registered first to intercept service registrations
	app.Register(rateLimitMiddleware)

	// Register regular modules
	app.Register(apiModule)

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	logger.Info("Rate Limiting Middleware Demo started")
	logger.Info("Services available",
		"services", []string{
			api.ServiceGetData + " (100 req/min)",
			api.ServiceCreateOrder + " (50 req/min)",
			api.ServiceGetStatus + " (200 req/min)",
		})
	logger.Info("Use nats CLI to test the services",
		"example", "nats request "+api.ServiceGetData+" '{}' --header X-Client-ID:client1")
	logger.Info("Press Ctrl+C to shutdown")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}

	logger.Info("Application stopped successfully")
}

// getEnv returns the environment variable value or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
