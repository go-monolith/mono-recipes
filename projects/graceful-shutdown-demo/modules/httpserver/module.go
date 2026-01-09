package httpserver

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
)

// HttpServerModule implements the mono Module interface for an HTTP server.
type HttpServerModule struct {
	app *fiber.App
}

// Compile-time interface check
var _ mono.HealthCheckableModule = (*HttpServerModule)(nil)

// Name returns the module name.
func (m *HttpServerModule) Name() string {
	return "http-server"
}

// Health performs a health check on the HTTP server module.
func (m *HttpServerModule) Health(_ context.Context) mono.HealthStatus {
	if m.app == nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: "HTTP server not initialized",
		}
	}
	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"port": 3000,
		},
	}
}

// Start initializes and starts the HTTP server.
func (m *HttpServerModule) Start(ctx context.Context) error {
	m.app = fiber.New()
	m.registerRoutes()

	return m.startServer(ctx)
}

// registerRoutes sets up all HTTP route handlers.
func (m *HttpServerModule) registerRoutes() {
	m.app.Get("/", m.handleRoot)
	m.app.Get("/health", m.handleHealth)
	m.app.Get("/slow", m.handleSlow)
}

func (m *HttpServerModule) handleRoot(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Hello from graceful-shutdown-demo!",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (m *HttpServerModule) handleHealth(c *fiber.Ctx) error {
	return c.JSON(m.Health(c.Context()))
}

// handleSlow demonstrates in-flight request handling during shutdown.
func (m *HttpServerModule) handleSlow(c *fiber.Ctx) error {
	log.Println("Slow request started...")

	select {
	case <-time.After(5 * time.Second):
		log.Println("Slow request completed!")
		return c.JSON(fiber.Map{
			"message": "This request took 5 seconds to complete",
			"time":    time.Now().Format(time.RFC3339),
		})
	case <-c.Context().Done():
		log.Println("Slow request cancelled")
		return c.Context().Err()
	}
}

// startServer starts the HTTP server in a goroutine and waits for it to be ready.
func (m *HttpServerModule) startServer(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		if err := m.app.Listen(":3000"); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("failed to start HTTP server: %w", err)
	case <-time.After(100 * time.Millisecond):
		log.Println("HTTP server started on :3000")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop gracefully shuts down the HTTP server.
func (m *HttpServerModule) Stop(ctx context.Context) error {
	if m.app == nil {
		return nil
	}

	log.Println("Shutting down HTTP server...")

	if err := m.app.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	log.Println("HTTP server stopped gracefully")
	return nil
}
