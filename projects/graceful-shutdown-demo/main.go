package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/graceful-shutdown-demo/modules/httpserver"
	"github.com/example/graceful-shutdown-demo/modules/worker"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("Starting graceful-shutdown-demo application...")

	// Create mono application with configuration
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
	)
	if err != nil {
		log.Fatalf("Failed to create mono application: %v", err)
	}

	// Register modules
	app.Register(&httpserver.HttpServerModule{})
	app.Register(&worker.WorkerModule{})

	// Start all modules
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Println("Application started successfully!")
	log.Println("Try these endpoints:")
	log.Println("  - http://localhost:3000/")
	log.Println("  - http://localhost:3000/health")
	log.Println("  - http://localhost:3000/slow (5 second delay)")
	log.Println("Press Ctrl+C to trigger graceful shutdown")

	// Setup graceful shutdown using gelmium/graceful-shutdown
	// This handles OS signals (SIGINT, SIGTERM, etc.)
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
