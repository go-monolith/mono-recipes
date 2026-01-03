package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/url-shortener-demo/modules/analytics"
	"github.com/example/url-shortener-demo/modules/api"
	"github.com/example/url-shortener-demo/modules/shortener"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== URL Shortener Demo - Fiber + JetStream KV Store ===")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register modules with the framework.
	// Order: independent modules first, then modules with dependencies
	// - analytics: Event consumer (subscribes to URL events)
	// - shortener: Core domain (JetStream KV storage, emits events)
	// - api: Driving adapter (Fiber HTTP server, depends on shortener)
	app.Register(analytics.NewModule()) // Event consumer
	app.Register(shortener.NewModule()) // Core domain + event emitter
	app.Register(api.NewModule())       // HTTP API

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	printStartupInfo()

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

func printStartupInfo() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	log.Println("")
	log.Println("Application started successfully!")
	log.Println("")
	log.Println("Architecture:")
	log.Println("  - HTTP Framework: Fiber")
	log.Println("  - Storage Backend: NATS JetStream KV Store")
	log.Printf("  - NATS URL: %s", natsURL)
	log.Printf("  - Base URL: %s", baseURL)
	log.Println("")
	log.Println("Event-Driven Analytics:")
	log.Println("  - URLCreated events -> analytics module")
	log.Println("  - URLAccessed events -> analytics module")
	log.Println("")
	log.Printf("REST API Endpoints (http://localhost:%s):", port)
	log.Println("  POST   /api/v1/shorten         - Shorten a URL")
	log.Println("  GET    /api/v1/stats/:code     - Get URL statistics")
	log.Println("  GET    /:code                  - Redirect to original URL")
	log.Println("  GET    /health                 - Health check")
	log.Println("")
	log.Println("Example: see demo.sh for curl commands to interact with the API")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
