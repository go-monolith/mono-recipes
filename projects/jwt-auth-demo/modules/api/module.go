package api

import (
	"context"
	"fmt"
	"log"

	"github.com/example/jwt-auth-demo/modules/auth"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// APIModule is the HTTP API module.
type APIModule struct {
	app           *fiber.App
	authContainer mono.ServiceContainer
	authAdapter   auth.AuthPort
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
func (m *APIModule) Dependencies() []string {
	return []string{"auth"}
}

// SetDependencyServiceContainer receives service containers from dependencies.
func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	switch dependency {
	case "auth":
		m.authContainer = container
		m.authAdapter = auth.NewAuthAdapter(container)
	}
}

// Start initializes the Fiber HTTP server.
func (m *APIModule) Start(_ context.Context) error {
	if m.authContainer == nil {
		return fmt.Errorf("auth dependency not set")
	}

	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler,
	})

	// Add middleware
	m.app.Use(recover.New())
	m.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	m.app.Use(cors.New())

	// Setup routes
	m.setupRoutes()

	// Start server in goroutine
	go func() {
		if err := m.app.Listen(":3000"); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Println("[api] HTTP server started on :3000")
	return nil
}

// Stop shuts down the Fiber HTTP server.
func (m *APIModule) Stop(_ context.Context) error {
	if m.app == nil {
		return nil
	}
	log.Println("[api] Shutting down HTTP server...")
	return m.app.Shutdown()
}

// Health returns the health status of the module.
func (m *APIModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"port": 3000,
		},
	}
}

// setupRoutes configures all API routes.
func (m *APIModule) setupRoutes() {
	handlers := NewHandlers(m.authContainer, m.authAdapter)

	// Health check endpoint
	m.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"module": "api",
		})
	})

	// API v1 routes
	v1 := m.app.Group("/api/v1")

	// Public auth routes
	authRoutes := v1.Group("/auth")
	authRoutes.Post("/register", handlers.Register)
	authRoutes.Post("/login", handlers.Login)
	authRoutes.Post("/refresh", handlers.Refresh)

	// Protected routes (require authentication)
	protected := v1.Group("")
	protected.Use(AuthMiddleware(m.authAdapter))
	protected.Get("/profile", handlers.Profile)
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
