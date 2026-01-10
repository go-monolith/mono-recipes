// http-fiber-module.go demonstrates HTTP server integration using Fiber framework
package httpserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Module implements HTTP server using Fiber framework
type Module struct {
	app    *fiber.App
	addr   string
	logger types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module         = (*Module)(nil)
	_ mono.DependentModule = (*Module)(nil)
)

// NewModule creates a new HTTP server module
func NewModule(addr string, logger types.Logger) *Module {
	return &Module{
		addr:   addr,
		logger: logger,
	}
}

// Name returns the module name
func (m *Module) Name() string { return "http-server" }

// Dependencies declares module dependencies
func (m *Module) Dependencies() []string {
	return []string{"user", "analytics"} // Example dependencies
}

// SetDependencyServiceContainer receives service containers from dependencies
func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
	// Store containers for handler use
	m.logger.Info("Received dependency container", "module", module)
}

// Start initializes and starts the HTTP server
func (m *Module) Start(ctx context.Context) error {
	// Create Fiber app with production config
	m.app = fiber.New(fiber.Config{
		AppName:               "My API",
		DisableStartupMessage: true,
		ErrorHandler:          m.errorHandler,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           60 * time.Second,
	})

	// Middleware stack
	m.app.Use(recover.New())
	m.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} ${method} ${path} ${latency}\n",
	}))

	// CORS configuration
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:8080"
	}
	m.app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	// Register routes
	m.registerRoutes()

	// Start server with startup error detection
	errCh := make(chan error, 1)
	go func() {
		if err := m.app.Listen(m.addr); err != nil {
			errCh <- err
		}
	}()

	// Wait briefly to catch immediate startup errors
	select {
	case err := <-errCh:
		return fmt.Errorf("HTTP server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	m.logger.Info("HTTP server started", "addr", m.addr)
	return nil
}

// Stop gracefully shuts down the HTTP server
func (m *Module) Stop(ctx context.Context) error {
	if m.app != nil {
		if err := m.app.ShutdownWithContext(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}
	m.logger.Info("HTTP server stopped")
	return nil
}

// registerRoutes sets up all HTTP routes
func (m *Module) registerRoutes() {
	// Health check
	m.app.Get("/health", m.handleHealth)

	// API routes
	api := m.app.Group("/api/v1")

	// Users resource
	users := api.Group("/users")
	users.Get("", m.handleListUsers)
	users.Get("/:id", m.handleGetUser)
	users.Post("", m.handleCreateUser)
	users.Put("/:id", m.handleUpdateUser)
	users.Delete("/:id", m.handleDeleteUser)
}

// Handlers
func (m *Module) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func (m *Module) handleListUsers(c *fiber.Ctx) error {
	// Call dependent module's service
	return c.JSON(fiber.Map{"users": []any{}})
}

func (m *Module) handleGetUser(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{"id": id})
}

func (m *Module) handleCreateUser(c *fiber.Ctx) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"created": true})
}

func (m *Module) handleUpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{"id": id, "updated": true})
}

func (m *Module) handleDeleteUser(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// errorHandler handles errors globally
func (m *Module) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	m.logger.Error("HTTP error", "code", code, "error", err)

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
