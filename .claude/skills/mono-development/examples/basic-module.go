// Example: Basic Module Implementation
//
// This example demonstrates the minimal module implementation
// with health checking capability.

package basicmodule

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
)

// ============================================================
// Module Implementation
// ============================================================

// HelloModule demonstrates the basic Module interface
type HelloModule struct {
	name string
}

// Compile-time interface checks
var (
	_ mono.Module                = (*HelloModule)(nil)
	_ mono.HealthCheckableModule = (*HelloModule)(nil)
)

// NewHelloModule creates a new HelloModule instance
func NewHelloModule() *HelloModule {
	return &HelloModule{
		name: "hello-world",
	}
}

// Name returns the unique module identifier
func (m *HelloModule) Name() string {
	return m.name
}

// Start initializes the module
func (m *HelloModule) Start(_ context.Context) error {
	slog.Info("Module started", "module", m.name)
	return nil
}

// Stop gracefully shuts down the module
func (m *HelloModule) Stop(_ context.Context) error {
	slog.Info("Module stopped", "module", m.name)
	return nil
}

// Health returns the current health status
func (m *HelloModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: true,
		Status:  "Running",
	}
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Basic Module Example ===")

	// Create application with configuration
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create and register module
	helloModule := NewHelloModule()
	if err := app.Register(helloModule); err != nil {
		log.Fatalf("Failed to register module: %v", err)
	}

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Printf("Registered modules: %v\n", app.Modules())

	// Check health
	health := app.Health(ctx)
	fmt.Printf("App Health: healthy=%v\n", health.Healthy)

	// Wait for shutdown signal
	fmt.Println("\nPress Ctrl+C to shutdown...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}

	fmt.Println("App stopped successfully")
}
