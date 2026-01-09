package api

import (
	"context"
	"fmt"
	"log"

	productmod "github.com/example/redis-caching-demo/modules/product"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Module provides the HTTP API for the caching demo.
type Module struct {
	app           *fiber.App
	handlers      *Handlers
	productModule *productmod.Module
	port          int
}

// Compile-time interface checks.
var _ mono.Module = (*Module)(nil)

// NewModule creates a new API module.
func NewModule(port int) *Module {
	return &Module{
		port: port,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "api"
}

// SetProductModule sets the product module dependency.
func (m *Module) SetProductModule(pm *productmod.Module) {
	m.productModule = pm
}

// Start initializes the Fiber app and starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	// Create Fiber app
	m.app = fiber.New(fiber.Config{
		AppName:               "Redis Caching Demo",
		DisableStartupMessage: true,
		ErrorHandler:          m.errorHandler,
	})

	// Global middleware
	m.app.Use(recover.New())
	m.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	m.app.Use(cors.New())

	// Create handlers with product service
	if m.productModule == nil {
		return fmt.Errorf("product module not set")
	}

	service := m.productModule.GetService()
	if service == nil {
		return fmt.Errorf("product service not available")
	}

	m.handlers = NewHandlers(service)

	// Setup routes
	m.setupRoutes()

	go func() {
		addr := fmt.Sprintf(":%d", m.port)
		log.Printf("[api] Starting HTTP server on %s", addr)
		if err := m.app.Listen(addr); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Println("[api] Module started")
	return nil
}

// setupRoutes configures all HTTP routes.
func (m *Module) setupRoutes() {
	// Health check
	m.app.Get("/health", m.handlers.HealthCheck)

	// API v1 routes
	api := m.app.Group("/api/v1")

	// Product CRUD endpoints
	products := api.Group("/products")
	products.Get("/", m.handlers.ListProducts)
	products.Get("/:id", m.handlers.GetProduct)
	products.Post("/", m.handlers.CreateProduct)
	products.Put("/:id", m.handlers.UpdateProduct)
	products.Delete("/:id", m.handlers.DeleteProduct)
}

// Stop stops the HTTP server gracefully.
func (m *Module) Stop(_ context.Context) error {
	if m.app != nil {
		log.Println("[api] Shutting down HTTP server...")
		if err := m.app.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
	}
	log.Println("[api] Module stopped")
	return nil
}

// errorHandler handles errors from Fiber routes.
func (m *Module) errorHandler(c *fiber.Ctx, err error) error {
	e, ok := err.(*fiber.Error)
	if !ok {
		e = fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}

	return c.Status(e.Code).JSON(fiber.Map{
		"error":  e.Message,
		"code":   e.Code,
		"path":   c.Path(),
		"method": c.Method(),
	})
}

// GetApp returns the Fiber app (for testing).
func (m *Module) GetApp() *fiber.App {
	return m.app
}
