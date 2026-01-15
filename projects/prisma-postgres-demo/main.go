package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/prisma-postgres-demo/modules/article"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== Prisma + sqlc PostgreSQL Demo ===")
	log.Println("Demonstrating Prisma migrations with sqlc type-safe queries")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register article module
	app.Register(article.NewModule())

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
	log.Print(`
Application started successfully!

This demo shows:
  - Prisma for local PostgreSQL development (PGlite)
  - Prisma migrations for schema management
  - sqlc type-safe SQL code generation
  - ServiceProviderModule pattern for request-reply services
  - No HTTP endpoints - pure service-based architecture

Available Services (via NATS request-reply):
  - services.article.create  - Create a new article
  - services.article.get     - Get article by ID or slug
  - services.article.list    - List articles with pagination
  - services.article.update  - Update article by ID
  - services.article.delete  - Delete article by ID
  - services.article.publish - Publish a draft article

Use the nats CLI to interact with services:
  nats request services.article.create '{"title":"Hello","content":"World","slug":"hello-world"}'

Run ./demo.sh to see full CRUD workflow

Press Ctrl+C to shutdown gracefully
`)
}
