// Package api provides REST API handlers for job management.
package api

import (
	"context"
	"fmt"
	"log"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
)

// Module provides the REST API as a mono module.
type Module struct {
	app             *fiber.App
	handler         *Handler
	service         *Service
	jobStore        *job.Store
	workerContainer mono.ServiceContainer
	port            int
}

// Compile-time interface checks.
var _ mono.Module = (*Module)(nil)
var _ mono.DependentModule = (*Module)(nil)

// NewModule creates a new API module.
func NewModule(port int, jobStore *job.Store) *Module {
	return &Module{
		port:     port,
		jobStore: jobStore,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "api"
}

// Dependencies declares that this module depends on the worker module.
func (m *Module) Dependencies() []string {
	return []string{"worker"}
}

// SetDependencyServiceContainer receives the worker module's service container.
func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
	if module == "worker" {
		m.workerContainer = container
	}
}

// Start initializes and starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	// Create Fiber app
	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_error",
				"message": "An unexpected error occurred",
			})
		},
	})

	// Create service and handler
	m.service = NewService(m.jobStore, m.workerContainer)
	m.handler = NewHandler(m.service)

	// Register routes
	m.handler.RegisterRoutes(m.app)

	// Health endpoint
	m.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "background-jobs-demo",
		})
	})

	// Start HTTP server in background
	go func() {
		addr := fmt.Sprintf(":%d", m.port)
		log.Printf("[api] Starting HTTP server on %s", addr)
		if err := m.app.Listen(addr); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Println("[api] API module started")
	return nil
}

// Stop stops the HTTP server gracefully.
func (m *Module) Stop(_ context.Context) error {
	if m.app != nil {
		if err := m.app.Shutdown(); err != nil {
			return err
		}
	}
	log.Println("[api] Module stopped")
	return nil
}

// GetApp returns the Fiber app instance.
func (m *Module) GetApp() *fiber.App {
	return m.app
}
