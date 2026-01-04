package wsserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/example/websocket-chat-demo/modules/chat"
)

// Module implements the WebSocket server module using Fiber framework.
type Module struct {
	app        *fiber.App
	handlers   *Handlers
	addr       string
	chatModule *chat.Module
	logger     types.Logger
}

// NewModule creates a new WebSocket server module.
func NewModule(addr string, chatModule *chat.Module, moduleLogger types.Logger) *Module {
	return &Module{
		addr:       addr,
		chatModule: chatModule,
		logger:     moduleLogger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "ws-server"
}

// Start initializes and starts the WebSocket server.
func (m *Module) Start(ctx context.Context) error {
	// Create Fiber app with custom config
	m.app = fiber.New(fiber.Config{
		AppName:               "WebSocket Chat Demo",
		DisableStartupMessage: true,
		ErrorHandler:          m.errorHandler,
	})

	// Add middleware
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
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	// Create handlers
	m.handlers = NewHandlers(m.chatModule)

	// Register routes
	m.registerRoutes()

	// Start server in goroutine with startup error detection
	errCh := make(chan error, 1)
	go func() {
		if err := m.app.Listen(m.addr); err != nil {
			errCh <- err
		}
	}()

	// Wait briefly to catch immediate startup errors
	select {
	case err := <-errCh:
		return fmt.Errorf("WebSocket server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	m.logger.Info("WebSocket server started", "addr", m.addr)
	return nil
}

// Stop gracefully shuts down the WebSocket server.
func (m *Module) Stop(ctx context.Context) error {
	if m.app != nil {
		if err := m.app.ShutdownWithContext(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}
	m.logger.Info("WebSocket server stopped")
	return nil
}

// registerRoutes sets up all HTTP and WebSocket routes.
func (m *Module) registerRoutes() {
	// Health check
	m.app.Get("/health", m.handlers.HealthCheck)

	// WebSocket upgrade middleware
	m.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket endpoint
	m.app.Get("/ws", websocket.New(m.handlers.HandleWebSocket))

	// REST API routes
	api := m.app.Group("/api/v1")
	api.Get("/rooms", m.handlers.ListRooms)
	api.Post("/rooms", m.handlers.CreateRoom)
	api.Get("/rooms/:id/history", m.handlers.GetRoomHistory)
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
