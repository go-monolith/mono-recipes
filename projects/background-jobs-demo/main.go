package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/api"
	"github.com/example/background-jobs-demo/modules/worker"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

func main() {
	log.Println("Starting background-jobs-demo application...")

	// Get API port from environment or use default
	apiPort := 8080
	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			apiPort = p
		} else {
			log.Printf("Warning: Invalid API_PORT '%s', using default %d: %v", port, apiPort, err)
		}
	}

	// Create shared job store
	jobStore := job.NewStore()

	// Create mono application with embedded NATS
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create and register modules
	// Worker module provides the QueueGroupService for job processing
	workerModule := worker.NewModule(jobStore)
	if err := app.Register(workerModule); err != nil {
		log.Fatalf("Failed to register worker module: %v", err)
	}

	// API module depends on worker module to send jobs
	apiModule := api.NewModule(apiPort, jobStore)
	if err := app.Register(apiModule); err != nil {
		log.Fatalf("Failed to register API module: %v", err)
	}

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Println("Application started successfully")
	log.Printf("API server listening on :%d", apiPort)
	log.Println("Using embedded NATS with QueueGroupService pattern")

	// Set up graceful shutdown
	shutdownTimeout := 30 * time.Second
	shutdownCtx, forceShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer forceShutdown()

	// Perform graceful shutdown
	shutdownChan := gfshutdown.GracefulShutdown(shutdownCtx, shutdownTimeout, map[string]gfshutdown.Operation{
		"application": func(ctx context.Context) error {
			return app.Stop(ctx)
		},
	})

	// Wait for shutdown to complete
	exitCode := <-shutdownChan
	if exitCode != 0 {
		log.Printf("Shutdown completed with exit code: %d", exitCode)
		os.Exit(exitCode)
	}

	log.Println("Shutdown completed successfully")
}
