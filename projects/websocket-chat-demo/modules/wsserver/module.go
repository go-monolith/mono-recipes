package wsserver

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	chatcontract "github.com/example/websocket-chat-demo/modules/chat/contract"
)

// Compile-time interface checks
var (
	_ mono.Module                 = (*Module)(nil)
	_ mono.DependentModule        = (*Module)(nil)
	_ mono.EventConsumerModule    = (*Module)(nil)
	_ types.ServiceConsumerModule = (*Module)(nil) // no mono.* alias in v0.1.1
)

// Module implements the WebSocket server module using Fiber framework. It is
// the driving adapter of the WebSocket and REST APIs and reaches the chat
// module only through ChatPort (ports.go) and the adapter over the chat
// contracts (adapter.go).
type Module struct {
	app      *fiber.App
	handlers *Handlers
	addr     string
	chat     ChatPort
	logger   types.Logger
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
	return []string{chatcontract.ModuleName}
}

// ConsumeServices declares every chat contract this module calls. The
// framework validates the list at startup against the chat module's services.
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		chatcontract.CreateRoom.ToBase(),
		chatcontract.GetRoom.ToBase(),
		chatcontract.ListRooms.ToBase(),
		chatcontract.JoinRoom.ToBase(),
		chatcontract.LeaveRoom.ToBase(),
		chatcontract.SendMessage.ToBase(),
		chatcontract.GetUser.ToBase(),
		chatcontract.GetHistory.ToBase(),
		chatcontract.GetRoomUsers.ToBase(),
	}
}

// SetDependencyServiceContainer wires the adapter behind ChatPort. The
// container is a read-only view of the chat module's services.
func (m *Module) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
	if dep == chatcontract.ModuleName {
		m.chat = chatAdapter{container: container}
		m.logger.Info("Received chat service container")
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "ws-server"
}

// RegisterEventConsumers subscribes to the chat events through the
// definitions published in chat/contract, to broadcast them to the WebSocket
// clients of the room.
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
	consumers := []struct {
		def     types.EventDefinition[chatcontract.ChatEvent]
		msgType string
	}{
		{chatcontract.UserJoinedV1, "user_joined"},
		{chatcontract.UserLeftV1, "user_left"},
		{chatcontract.ChatMessageV1, "chat_message"},
	}
	for _, c := range consumers {
		if err := helper.RegisterTypedEventConsumer(registry, c.def, m.broadcast(c.msgType), m); err != nil {
			return fmt.Errorf("failed to register %s consumer: %w", c.def.Name, err)
		}
	}

	m.logger.Info("Registered WebSocket event consumers")
	return nil
}

// broadcast returns an event handler that forwards the event's message to the
// WebSocket clients in the event's room as msgType.
func (m *Module) broadcast(msgType string) func(context.Context, chatcontract.ChatEvent, *mono.Msg) error {
	return func(_ context.Context, event chatcontract.ChatEvent, _ *mono.Msg) error {
		if m.handlers != nil {
			m.handlers.BroadcastToRoom(event.RoomID, msgType, event.Message)
		}
		return nil
	}
}

// Start initializes and starts the WebSocket server.
func (m *Module) Start(ctx context.Context) error {
	if m.chat == nil {
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

	// Create handlers with the chat port
	m.handlers = NewHandlers(m.chat)

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
