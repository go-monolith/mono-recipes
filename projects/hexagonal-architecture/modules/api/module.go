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

var _ mono.Module = (*APIModule)(nil)
var _ mono.DependentModule = (*APIModule)(nil)
var _ mono.HealthCheckableModule = (*APIModule)(nil)

func NewModule() *APIModule {
	return &APIModule{}
}

func (m *APIModule) Name() string {
	return "api"
}

func (m *APIModule) Dependencies() []string {
	return []string{"task"}
}

func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == "task" {
		m.taskAdapter = task.NewTaskAdapter(container)
	}
}

func (m *APIModule) Start(_ context.Context) error {
	if m.taskAdapter == nil {
		return fmt.Errorf("taskAdapter dependency not set")
	}

	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler,
	})
	m.app.Use(recover.New())
	m.setupRoutes()

	go func() {
		if err := m.app.Listen(":3000"); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Println("[api] HTTP server started on :3000")
	return nil
}

func (m *APIModule) Stop(_ context.Context) error {
	if m.app == nil {
		return nil
	}
	log.Println("[api] Shutting down HTTP server...")
	return m.app.Shutdown()
}

func (m *APIModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"port": 3000,
		},
	}
}

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
