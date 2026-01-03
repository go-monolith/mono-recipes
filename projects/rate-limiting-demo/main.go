// Rate Limiting Demo - A demonstration of distributed rate limiting using Fiber and Redis.
//
// This application showcases:
// - Sliding window rate limiting algorithm
// - Per-IP rate limiting for public endpoints
// - Per-API-key rate limiting for premium endpoints
// - Redis-based distributed rate limiting for horizontal scaling
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/example/rate-limiting-demo/modules/api"
	"github.com/example/rate-limiting-demo/modules/ratelimit"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== Rate Limiting Demo - Fiber + Redis ===")

	// Configuration from environment variables with defaults
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpPort := getEnvInt("HTTP_PORT", 8080)

	log.Printf("Configuration:")
	log.Printf("  Redis Address: %s", redisAddr)
	log.Printf("  HTTP Port: %d", httpPort)

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create modules
	rateLimitModule := ratelimit.NewModule(redisAddr)
	apiModule := api.NewModule(httpPort)

	// Inject dependencies
	apiModule.SetRateLimitModule(rateLimitModule)

	// Register modules (order matters: rate limit module first)
	app.Register(rateLimitModule)
	app.Register(apiModule)

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	printStartupInfo(httpPort)

	// Graceful shutdown
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

	exitCode := <-wait
	log.Printf("Application exited with code: %d", exitCode)
	os.Exit(exitCode)
}

func printStartupInfo(port int) {
	log.Println("")
	log.Println("Application started successfully!")
	log.Println("")
	log.Println("Architecture:")
	log.Println("  - HTTP Framework: Fiber")
	log.Println("  - Rate Limiting: Sliding Window Algorithm")
	log.Println("  - Storage Backend: Redis")
	log.Println("")
	log.Println("Rate Limits:")
	log.Println("  - Public endpoint: 100 requests per minute (by IP)")
	log.Println("  - Premium endpoint: 1000 requests per minute (by API key)")
	log.Println("")
	log.Printf("REST API Endpoints (http://localhost:%d):", port)
	log.Println("  GET  /health         - Health check (no rate limiting)")
	log.Println("  GET  /api/v1/public  - Public endpoint (IP-based rate limit)")
	log.Println("  GET  /api/v1/premium - Premium endpoint (API key rate limit)")
	log.Println("  GET  /api/v1/stats   - Rate limit statistics")
	log.Println("")
	log.Println("Example: see demo.sh for commands to test rate limiting")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable or a default value.
// Logs a warning if the value cannot be parsed as an integer.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
		log.Printf("Warning: invalid integer value for %s: %q, using default %d", key, value, defaultValue)
	}
	return defaultValue
}
