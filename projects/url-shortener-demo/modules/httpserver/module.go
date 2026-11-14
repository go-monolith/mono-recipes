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

	analyticscontract "github.com/example/url-shortener-demo/modules/analytics/contract"
	shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"
)

// Module implements the HTTP server module using Fiber framework. It is the
// driving adapter of the application's HTTP API and reaches the shortener and
// analytics modules only through its own ports (ports.go) and the adapters
// over their contracts (adapter.go).
type Module struct {
	app       *fiber.App
	handlers  *Handlers
	addr      string
	shortener ShortenerPort
	analytics AnalyticsPort
	logger    types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                 = (*Module)(nil)
	_ mono.DependentModule        = (*Module)(nil)
	_ types.ServiceConsumerModule = (*Module)(nil) // no mono.* alias in v0.1.1
)

// NewModule creates a new HTTP server module.
func NewModule(addr string, moduleLogger types.Logger) *Module {
	return &Module{
		addr:   addr,
		logger: moduleLogger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "http-server"
}

// Dependencies declares which modules this module depends on.
func (m *Module) Dependencies() []string {
	return []string{shortenercontract.ModuleName, analyticscontract.ModuleName}
}

// ConsumeServices declares every contract this module calls. The framework
// validates the list at startup: each provider must be a declared dependency,
// must have registered the service, and the service kind must match.
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		shortenercontract.ShortenURL.ToBase(),
		shortenercontract.ResolveURL.ToBase(),
		shortenercontract.GetStats.ToBase(),
		shortenercontract.ListURLs.ToBase(),
		shortenercontract.DeleteURL.ToBase(),
		analyticscontract.GetSummary.ToBase(),
		analyticscontract.GetLogs.ToBase(),
	}
}

// SetDependencyServiceContainer wires the adapters behind the ports. The
// container is a read-only view of the dependency's services.
func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
	switch module {
	case shortenercontract.ModuleName:
		m.shortener = shortenerAdapter{container: container}
		m.logger.Info("Received shortener service container")
	case analyticscontract.ModuleName:
		m.analytics = analyticsAdapter{container: container}
		m.logger.Info("Received analytics service container")
	}
}

// Start initializes and starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	if m.shortener == nil || m.analytics == nil {
		return fmt.Errorf("shortener and analytics dependencies not set - dependency injection failed")
	}

	// Create Fiber app with custom config
	m.app = fiber.New(fiber.Config{
		AppName:               "URL Shortener Demo",
		DisableStartupMessage: true,
		ErrorHandler:          m.errorHandler,
	})

	// Add middleware
	m.app.Use(recover.New())
	m.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} ${method} ${path} ${latency}\n",
	}))
	// CORS configuration - restrict to specific origins in production
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:8080"
	}
	m.app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	// Create handlers
	m.handlers = NewHandlers(m.shortener, m.analytics)

	// Register routes
	m.registerRoutes()

	// Start server in goroutine with startup error detection
	errCh := make(chan error, 1)
	go func() {
		if err := m.app.Listen(m.addr); err != nil {
			errCh <- err
		}
	}()

	// Wait briefly to catch immediate startup errors (port in use, permission denied)
	select {
	case err := <-errCh:
		return fmt.Errorf("HTTP server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	m.logger.Info("HTTP server started", "addr", m.addr)
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (m *Module) Stop(ctx context.Context) error {
	if m.app != nil {
		if err := m.app.ShutdownWithContext(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}
	m.logger.Info("HTTP server stopped")
	return nil
}

// registerRoutes sets up all HTTP routes.
func (m *Module) registerRoutes() {
	// Health check
	m.app.Get("/health", m.handlers.HealthCheck)

	// API routes
	api := m.app.Group("/api/v1")

	// URL shortening
	api.Post("/shorten", m.handlers.ShortenURL)

	// URL management
	api.Get("/urls", m.handlers.ListURLs)
	api.Delete("/urls/:shortCode", m.handlers.DeleteURL)

	// Statistics
	api.Get("/stats/:shortCode", m.handlers.GetStats)

	// Analytics
	api.Get("/analytics", m.handlers.GetAnalytics)
	api.Get("/analytics/logs", m.handlers.GetAnalyticsLogs)

	// Redirect (must be last to avoid conflicts)
	m.app.Get("/:shortCode", m.handlers.Redirect)
}

// errorHandler handles errors globally.
func (m *Module) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	m.logger.Error("HTTP error", "code", code, "message", message, "error", err)

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
