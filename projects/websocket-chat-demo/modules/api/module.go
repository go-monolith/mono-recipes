package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/example/websocket-chat-demo/modules/broadcast"
	"github.com/example/websocket-chat-demo/modules/chat"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// APIModule is the HTTP API module with WebSocket support.
type APIModule struct {
	app         *fiber.App
	chatAdapter chat.ChatPort
	hub         *broadcast.Hub
	port        string
}

// Compile-time interface checks.
var _ mono.Module = (*APIModule)(nil)
var _ mono.DependentModule = (*APIModule)(nil)
var _ mono.HealthCheckableModule = (*APIModule)(nil)

// NewModule creates a new APIModule.
func NewModule() *APIModule {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return &APIModule{
		port: port,
	}
}

// Name returns the module name.
func (m *APIModule) Name() string {
	return "api"
}

// Dependencies returns the list of module dependencies.
func (m *APIModule) Dependencies() []string {
	return []string{"chat"}
}

// SetDependencyServiceContainer receives service containers from dependencies.
func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	switch dependency {
	case "chat":
		m.chatAdapter = chat.NewChatAdapter(container)
	}
}

// SetHub sets the broadcast hub (called from main.go).
func (m *APIModule) SetHub(hub *broadcast.Hub) {
	m.hub = hub
}

// Start initializes the Fiber HTTP server.
func (m *APIModule) Start(_ context.Context) error {
	if m.chatAdapter == nil {
		return fmt.Errorf("chat adapter dependency not set")
	}
	if m.hub == nil {
		return fmt.Errorf("broadcast hub dependency not set")
	}

	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          60 * time.Second,
		IdleTimeout:           120 * time.Second,
	})

	// Add recovery middleware
	m.app.Use(recover.New())

	// Add logging middleware
	m.app.Use(loggerMiddleware())

	// Setup routes
	m.setupRoutes()

	// Start server in goroutine
	go func() {
		if err := m.app.Listen(":" + m.port); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Printf("[api] HTTP server started on :%s", m.port)
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

// Health returns the health status.
func (m *APIModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"port":              m.port,
			"connected_clients": m.hub.ClientCount(),
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

// loggerMiddleware returns a Fiber middleware for request logging.
func loggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip logging for WebSocket upgrade requests
		if c.Get("Upgrade") == "websocket" {
			return c.Next()
		}
		err := c.Next()
		log.Printf("[api] %s %s %d", c.Method(), c.Path(), c.Response().StatusCode())
		return err
	}
}
