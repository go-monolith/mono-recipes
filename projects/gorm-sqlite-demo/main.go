package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/gorm-sqlite-demo/modules/product"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== GORM + SQLite Demo ===")
	log.Println("Demonstrating ORM-based database integration with mono framework")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register product module
	// The framework automatically calls:
	// - ServiceProviderModule.RegisterServices() for request-reply services
	app.Register(product.NewModule())

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
	log.Println("")
	log.Println("Application started successfully!")
	log.Println("")
	log.Println("This demo shows:")
	log.Println("  - GORM ORM integration with SQLite")
	log.Println("  - ServiceProviderModule pattern for request-reply services")
	log.Println("  - Automatic database migration on startup")
	log.Println("  - No HTTP endpoints - pure service-based architecture")
	log.Println("")
	log.Println("Available Services (via NATS request-reply):")
	log.Println("  - product.create  - Create a new product")
	log.Println("  - product.get     - Get product by ID")
	log.Println("  - product.list    - List all products")
	log.Println("  - product.update  - Update product by ID")
	log.Println("  - product.delete  - Delete product by ID")
	log.Println("")
	log.Println("Use the nats CLI to interact with services:")
	log.Println("  nats request services.product.create '{\"name\":\"Widget\",\"price\":9.99}'")
	log.Println("")
	log.Println("Run ./demo.sh to see full CRUD workflow")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
