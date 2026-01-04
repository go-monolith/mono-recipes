package api

import (
	"context"
	"fmt"
	"log"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/nats"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
)

// Module provides the REST API as a mono module.
type Module struct {
	app        *fiber.App
	handler    *Handler
	service    *Service
	jobStore   *job.Store
	natsClient *nats.Client
	port       int
}

// NewModule creates a new API module.
func NewModule(port int, jobStore *job.Store, natsClient *nats.Client) *Module {
	return &Module{
		port:       port,
		jobStore:   jobStore,
		natsClient: natsClient,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "api"
}

// Init initializes the API module.
func (m *Module) Init(_ mono.ServiceContainer) error {
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
	m.service = NewService(m.jobStore, m.natsClient)
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

	log.Println("[api] API module initialized")
	return nil
}

// Start starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	go func() {
		addr := fmt.Sprintf(":%d", m.port)
		log.Printf("[api] Starting HTTP server on %s", addr)
		if err := m.app.Listen(addr); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()
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
