package wsserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/example/websocket-chat-demo/modules/chat"
)

// Compile-time interface checks
var (
	_ mono.Module              = (*Module)(nil)
	_ mono.DependentModule     = (*Module)(nil)
	_ mono.EventConsumerModule = (*Module)(nil)
)

// Module implements the WebSocket server module using Fiber framework.
type Module struct {
	app          *fiber.App
	handlers     *Handlers
	addr         string
	chatAdapter  *chat.ServiceAdapter
	logger       types.Logger
}

// NewModule creates a new WebSocket server module.
func NewModule(addr string, moduleLogger types.Logger) *Module {
	return &Module{
		addr:   addr,
		logger: moduleLogger,
	}
}

// Dependencies declares the modules this module depends on.
func (m *Module) Dependencies() []string {
	return []string{"chat"}
}

// SetDependencyServiceContainer receives the service container from a dependency.
func (m *Module) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
	if dep == "chat" {
		m.chatAdapter = chat.NewServiceAdapter(container)
		m.logger.Info("Received chat service container")
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "ws-server"
}

// RegisterEventConsumers registers event handlers for chat broadcasts.
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Register consumer for UserJoined events
	userJoinedDef, ok := registry.GetEventByName("UserJoined", "v1", "chat")
	if !ok {
		return fmt.Errorf("event UserJoined.v1 not found")
	}
	if err := registry.RegisterEventConsumer(userJoinedDef, m.handleUserJoinedEvent, m); err != nil {
		return fmt.Errorf("failed to register UserJoined consumer: %w", err)
	}

	// Register consumer for UserLeft events
	userLeftDef, ok := registry.GetEventByName("UserLeft", "v1", "chat")
	if !ok {
		return fmt.Errorf("event UserLeft.v1 not found")
	}
	if err := registry.RegisterEventConsumer(userLeftDef, m.handleUserLeftEvent, m); err != nil {
		return fmt.Errorf("failed to register UserLeft consumer: %w", err)
	}

	// Register consumer for ChatMessage events
	chatMsgDef, ok := registry.GetEventByName("ChatMessage", "v1", "chat")
	if !ok {
		return fmt.Errorf("event ChatMessage.v1 not found")
	}
	if err := registry.RegisterEventConsumer(chatMsgDef, m.handleChatMessageEvent, m); err != nil {
		return fmt.Errorf("failed to register ChatMessage consumer: %w", err)
	}

	m.logger.Info("Registered WebSocket event consumers")
	return nil
}

// handleUserJoinedEvent broadcasts user join to WebSocket clients.
func (m *Module) handleUserJoinedEvent(_ context.Context, msg *mono.Msg) error {
	var event chat.ChatEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		m.logger.Error("Failed to unmarshal UserJoined event", "error", err)
		return nil
	}

	if m.handlers != nil {
		m.handlers.BroadcastToRoom(event.RoomID, "user_joined", event.Message)
	}
	return nil
}

// handleUserLeftEvent broadcasts user leave to WebSocket clients.
func (m *Module) handleUserLeftEvent(_ context.Context, msg *mono.Msg) error {
	var event chat.ChatEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		m.logger.Error("Failed to unmarshal UserLeft event", "error", err)
		return nil
	}

	if m.handlers != nil {
		m.handlers.BroadcastToRoom(event.RoomID, "user_left", event.Message)
	}
	return nil
}

// handleChatMessageEvent broadcasts chat message to WebSocket clients.
func (m *Module) handleChatMessageEvent(_ context.Context, msg *mono.Msg) error {
	var event chat.ChatEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		m.logger.Error("Failed to unmarshal ChatMessage event", "error", err)
		return nil
	}

	if m.handlers != nil {
		m.handlers.BroadcastToRoom(event.RoomID, "chat_message", event.Message)
	}
	return nil
}

// Start initializes and starts the WebSocket server.
func (m *Module) Start(ctx context.Context) error {
	if m.chatAdapter == nil {
		return fmt.Errorf("chat adapter not set - dependency injection failed")
	}

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

	// Create handlers with chat adapter
	m.handlers = NewHandlers(m.chatAdapter)

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
