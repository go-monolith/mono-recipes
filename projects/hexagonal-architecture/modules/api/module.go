package api

import (
	"context"
	"fmt"
	"log"

	"github.com/example/hexagonal-architecture/modules/task"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// APIModule is the driving adapter that exposes REST endpoints.
// It calls into the core domain (task module) via the TaskPort interface.
type APIModule struct {
	app         *fiber.App
	taskAdapter task.TaskPort
}

// Compile-time interface checks.
var _ mono.Module = (*APIModule)(nil)
var _ mono.DependentModule = (*APIModule)(nil)
var _ mono.HealthCheckableModule = (*APIModule)(nil)

// NewModule creates a new APIModule.
func NewModule() *APIModule {
	return &APIModule{}
}

// Name returns the module name.
func (m *APIModule) Name() string {
	return "api"
}

// Dependencies returns the list of module dependencies.
// The framework will call SetDependencyServiceContainer for each dependency.
func (m *APIModule) Dependencies() []string {
	return []string{"task"}
}

// SetDependencyServiceContainer receives service containers from dependencies.
// This is called by the framework for each dependency declared in Dependencies().
func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	switch dependency {
	case "task":
		m.taskAdapter = task.NewTaskAdapter(container)
	}
}

// Start initializes the Fiber HTTP server.
// Returns an error if required dependencies are not set.
func (m *APIModule) Start(_ context.Context) error {
	if m.taskAdapter == nil {
		return fmt.Errorf("taskAdapter dependency not set")
	}

	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler,
	})

	// Add recovery middleware
	m.app.Use(recover.New())

	// Setup routes
	m.setupRoutes()

	// Start server in goroutine.
	// Server availability is verified via Health() method.
	go func() {
		if err := m.app.Listen(":3000"); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Println("[api] HTTP server started on :3000")
	return nil
}

// Stop shuts down the Fiber HTTP server.
func (m *APIModule) Stop(ctx context.Context) error {
	if m.app == nil {
		return nil
	}
	log.Println("[api] Shutting down HTTP server...")
	return m.app.Shutdown()
}

// Health returns the health status of the module.
func (m *APIModule) Health(ctx context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"port": 3000,
		},
	}
}

// customErrorHandler handles Fiber errors.
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(ErrorResponse{
		Error:   "server_error",
		Message: message,
	})
}
