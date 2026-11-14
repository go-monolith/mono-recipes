package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	fileservicemod "github.com/example/file-upload-demo/modules/fileservice"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
)

const shutdownTimeout = 30 * time.Second

func main() {
	// Load configuration from environment
	httpPort := getEnvInt("HTTP_PORT", 3000)
	maxUploadSize := getEnvInt64("MAX_UPLOAD_SIZE", 100*1024*1024) // 100MB default
	storagePath := getEnv("STORAGE_PATH", "/tmp/file-upload-demo")

	log.Println("=== File Upload Demo ===")
	log.Printf("HTTP Port: %d", httpPort)
	log.Printf("Max Upload Size: %d bytes", maxUploadSize)
	log.Printf("Storage Path: %s", storagePath)

	// Create mono application with embedded NATS JetStream
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithJetStreamStorageDir(storagePath),
		// Module boundary violations fail instead of only being reported at Stop.
		mono.WithStrictModuleBoundaries(),
	)
	if err != nil {
		log.Fatalf("Failed to create mono application: %v", err)
	}

	// Create fs-jetstream plugin for file storage
	// This uses the embedded NATS server - no external NATS required
	storagePlugin, err := fsjetstream.New(fsjetstream.Config{
		Buckets: []fsjetstream.BucketConfig{
			{
				Name:        "files",
				Description: "File storage bucket",
				MaxBytes:    1024 * 1024 * 1024, // 1GB max storage
				Storage:     fsjetstream.FileStorage,
				Compression: true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create storage plugin: %v", err)
	}

	// Register storage plugin with alias "storage"
	// The framework will call SetPlugin("storage", storagePlugin) on modules
	// that implement UsePluginModule interface
	if err := app.RegisterPlugin(storagePlugin, "storage"); err != nil {
		log.Fatalf("Failed to register storage plugin: %v", err)
	}

	// Create the file service module. It owns both the storage service and the
	// Gin HTTP API that drives it: uploads and downloads are binary payloads
	// handed to the storage service in-process, so there is no second module
	// to wire up with a direct reference.
	fileServiceModule := fileservicemod.NewModule(httpPort, maxUploadSize, app.Logger())

	// Register the module. It implements UsePluginModule and will receive the
	// storage plugin before Start.
	if err := app.Register(fileServiceModule); err != nil {
		log.Fatalf("Failed to register module: %v", err)
	}

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	log.Println("=== Application Started ===")
	log.Printf("API available at http://localhost:%d", httpPort)
	log.Println("Endpoints:")
	log.Println("  GET    /health                   - Health check")
	log.Println("  POST   /api/v1/files             - Upload a file")
	log.Println("  POST   /api/v1/files/batch       - Upload multiple files")
	log.Println("  GET    /api/v1/files             - List all files")
	log.Println("  GET    /api/v1/files/:id         - Download a file")
	log.Println("  GET    /api/v1/files/:id/info    - Get file metadata")
	log.Println("  DELETE /api/v1/files/:id         - Delete a file")
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

// getEnvInt64 returns environment variable as int64 or default.
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid int64 value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}
