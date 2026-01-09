package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/sqlc-postgres-demo/modules/user"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== sqlc + PostgreSQL Demo ===")
	log.Println("Demonstrating type-safe SQL with code generation")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register user module
	// The framework automatically calls:
	// - ServiceProviderModule.RegisterServices() for request-reply services
	app.Register(user.NewModule())

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
	log.Println("  - sqlc type-safe SQL code generation")
	log.Println("  - PostgreSQL database integration")
	log.Println("  - ServiceProviderModule pattern for request-reply services")
	log.Println("  - Pagination support for list operations")
	log.Println("  - No HTTP endpoints - pure service-based architecture")
	log.Println("")
	log.Println("Available Services (via NATS request-reply):")
	log.Println("  - user.create  - Create a new user")
	log.Println("  - user.get     - Get user by ID")
	log.Println("  - user.list    - List users with pagination")
	log.Println("  - user.update  - Update user by ID")
	log.Println("  - user.delete  - Delete user by ID")
	log.Println("")
	log.Println("Use the nats CLI to interact with services:")
	log.Println("  nats request services.user.create '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
	log.Println("")
	log.Println("Run ./demo.sh to see full CRUD workflow with psql verification")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
