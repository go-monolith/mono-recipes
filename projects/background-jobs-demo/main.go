package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/api"
	"github.com/example/background-jobs-demo/modules/eventbus"
	"github.com/example/background-jobs-demo/modules/nats"
	"github.com/example/background-jobs-demo/modules/worker"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

func main() {
	log.Println("Starting background-jobs-demo application...")

	// Get NATS URL from environment or use default
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Get API port from environment or use default
	apiPort := 8080
	if port := os.Getenv("API_PORT"); port != "" {
		log.Printf("Using API_PORT from environment: %s", port)
	}

	// Create shared job store
	jobStore := job.NewStore()

	// Create NATS client module
	natsModule := nats.NewModule(natsURL)

	// Create EventBus module
	eventBusModule := eventbus.NewModule()

	// Get NATS client after initialization (we'll access it after app init)
	var natsClient *nats.Client
	var eventBus *eventbus.EventBus

	// Create mono application
	app, err := mono.NewMonoApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register modules
	app.Register(natsModule)
	app.Register(eventBusModule)

	// Get module instances (they'll be initialized when app starts)
	natsClient = natsModule.GetClient()
	eventBus = eventBusModule.GetEventBus()

	// Create worker pool module with configuration
	workerConfig := worker.PoolConfig{
		NumWorkers:     3,
		MaxRetries:     5,
		BaseRetryDelay: time.Second,
		MaxRetryDelay:  time.Minute,
		ProcessTimeout: 5 * time.Minute,
	}
	workerModule := worker.NewModuleWithConfig(workerConfig, natsClient, eventBus, jobStore)

	// Create API module
	apiModule := api.NewModule(apiPort, jobStore, natsClient)

	// Register worker and API modules
	app.Register(workerModule)
	app.Register(apiModule)

	// Subscribe to job events for logging
	eventBus.SubscribeAll(func(event job.JobEvent) {
		switch event.Type {
		case job.EventTypeJobStarted:
			if data, ok := event.Data.(job.JobStartedData); ok {
				log.Printf("[event] Job started: %s (type=%s, worker=%s)", event.JobID, event.JobType, data.WorkerID)
			}
		case job.EventTypeJobProgress:
			if data, ok := event.Data.(job.JobProgressData); ok {
				log.Printf("[event] Job progress: %s (progress=%d%%, message=%s)", event.JobID, data.Progress, data.Message)
			}
		case job.EventTypeJobCompleted:
			if data, ok := event.Data.(job.JobCompletedData); ok {
				log.Printf("[event] Job completed: %s (type=%s, duration=%v)", event.JobID, event.JobType, data.Duration)
			}
		case job.EventTypeJobFailed:
			if data, ok := event.Data.(job.JobFailedData); ok {
				log.Printf("[event] Job failed: %s (type=%s, error=%s, retry=%d, will_retry=%v)",
					event.JobID, event.JobType, data.Error, data.RetryCount, data.WillRetry)
			}
		case job.EventTypeJobDeadLetter:
			if data, ok := event.Data.(job.JobDeadLetterData); ok {
				log.Printf("[event] Job moved to dead-letter queue: %s (type=%s, reason=%s, retries=%d)",
					event.JobID, event.JobType, data.Reason, data.RetryCount)
			}
		}
	})

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Println("Application started successfully")
	log.Printf("API server listening on :%d", apiPort)
	log.Printf("Worker pool running with %d workers", workerConfig.NumWorkers)
	log.Printf("NATS connected to %s", natsURL)

	// Set up graceful shutdown
	shutdownTimeout := 30 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\nReceived shutdown signal, shutting down gracefully...")

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
