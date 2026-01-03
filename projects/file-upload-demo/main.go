package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/file-upload-demo/modules/api"
	"github.com/example/file-upload-demo/modules/files"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== File Upload Demo - Gin + JetStream Object Store ===")

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
	app.Register(files.NewModule()) // Core domain (JetStream storage)
	app.Register(api.NewModule())   // Driving adapter (Gin HTTP server)

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

	log.Println("")
	log.Println("Application started successfully!")
	log.Println("")
	log.Println("Architecture:")
	log.Println("  - HTTP Framework: Gin")
	log.Println("  - Storage Backend: NATS JetStream Object Store")
	log.Printf("  - NATS URL: %s", natsURL)
	log.Println("")
	log.Printf("REST API Endpoints (http://localhost:%s):", port)
	log.Println("  POST   /api/v1/files           - Upload a file (multipart/form-data or JSON)")
	log.Println("  GET    /api/v1/files           - List all files")
	log.Println("  GET    /api/v1/files/:id       - Get file metadata")
	log.Println("  GET    /api/v1/files/:id/download - Download file content")
	log.Println("  DELETE /api/v1/files/:id       - Delete a file")
	log.Println("  GET    /health                 - Health check")
	log.Println("")
	log.Println("Example: see demo.sh for curl commands to interact with the API")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
