package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"node-nats-client-demo/modules/fileops"

	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
)

const shutdownTimeout = 30 * time.Second

func main() {
	// Load configuration from environment
	storagePath := getEnv("STORAGE_PATH", "/tmp/node-nats-client-demo")
	natsPort := getEnvInt("NATS_PORT", 4222)

	log.Println("=== Node.js NATS Client Demo ===")
	log.Printf("Storage Path: %s", storagePath)
	log.Printf("NATS Port: %d", natsPort)
	log.Println("Using embedded NATS server (no external dependencies)")

	// Create mono application with embedded NATS JetStream
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithJetStreamStorageDir(storagePath),
		mono.WithNATSPort(natsPort),
	)
	if err != nil {
		log.Fatalf("Failed to create mono application: %v", err)
	}

	// Create fs-jetstream plugin for file storage
	storagePlugin, err := fsjetstream.New(fsjetstream.Config{
		Buckets: []fsjetstream.BucketConfig{
			{
				Name:        "user-settings",
				Description: "User settings storage bucket",
				MaxBytes:    500 * 1024 * 1024, // 500MB max storage
				Storage:     fsjetstream.FileStorage,
				Compression: true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create storage plugin: %v", err)
	}

	// Register storage plugin with alias "storage"
	if err := app.RegisterPlugin(storagePlugin, "storage"); err != nil {
		log.Fatalf("Failed to register storage plugin: %v", err)
	}

	// Create and register file operations module
	fileOpsModule := fileops.NewModule(app.Logger())
	app.Register(fileOpsModule)

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	log.Println("=== Application Started ===")
	log.Printf("NATS available at nats://localhost:%d", natsPort)
	log.Println("Services:")
	log.Println("  services.fileops.save    - RequestReplyService: Save JSON file to bucket")
	log.Println("  services.fileops.archive - QueueGroupService: Archive JSON file as ZIP")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown")

	// Setup graceful shutdown
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

	// Wait for shutdown signal
	exitCode := <-wait
	log.Printf("Application exited with code: %d", exitCode)
	os.Exit(exitCode)
}

// getEnv returns environment variable value or default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid int value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}
