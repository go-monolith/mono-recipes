package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/hexagonal-architecture/modules/api"
	"github.com/example/hexagonal-architecture/modules/notification"
	"github.com/example/hexagonal-architecture/modules/task"
	"github.com/example/hexagonal-architecture/modules/user"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== Hexagonal Architecture Demo - Task/Todo System ===")

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
	// The framework automatically handles:
	// - ServiceProviderModule.RegisterServices() for request-reply services
	// - DependentModule.SetDependencyServiceContainer() for cross-module communication
	// - EventBusAwareModule.SetEventBus() for event publishing
	// - EventConsumerModule.RegisterEventConsumers() for event subscriptions
	//
	// Order: independent modules first, then modules with dependencies
	app.Register(user.NewModule())         // Independent module (no dependencies)
	app.Register(notification.NewModule()) // Event consumer (subscribes to task events)
	app.Register(task.NewModule())         // Core domain (depends on user, emits events)
	app.Register(api.NewModule())          // Driving adapter (depends on task)

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
	log.Println("Hexagonal Architecture Patterns Demonstrated:")
	log.Println("  - Driving Adapter: API module (Fiber HTTP server)")
	log.Println("  - Core Domain: Task module (business logic)")
	log.Println("  - Driven Adapter: Notification module (event consumer)")
	log.Println("  - Ports: TaskPort, UserPort interfaces")
	log.Println("")
	log.Println("Demo Users Available:")
	log.Println("  - user-1: Alice Johnson (alice@example.com)")
	log.Println("  - user-2: Bob Smith (bob@example.com)")
	log.Println("  - user-3: Charlie Brown (charlie@example.com)")
	log.Println("")
	log.Println("REST API Endpoints (http://localhost:3000):")
	log.Println("  POST   /api/v1/tasks           - Create a task")
	log.Println("  GET    /api/v1/tasks           - List all tasks")
	log.Println("  GET    /api/v1/tasks/:id       - Get a task by ID")
	log.Println("  PUT    /api/v1/tasks/:id       - Update a task")
	log.Println("  DELETE /api/v1/tasks/:id       - Delete a task")
	log.Println("  POST   /api/v1/tasks/:id/complete - Complete a task")
	log.Println("  GET    /health                 - Health check")
	log.Println("")
	log.Println("Example: see demo.sh for curl commands to interact with the API")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
