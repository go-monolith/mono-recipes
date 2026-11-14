// Package api is the HTTP driving adapter of the demo (Fiber). It reaches the
// auth module only through the AuthPort it owns, implemented over
// auth/contract; it imports nothing else of the auth module.
package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

// DefaultAddr is the address the HTTP server listens on.
const DefaultAddr = ":3000"

// APIModule is the HTTP API module.
type APIModule struct {
	addr string
	app  *fiber.App
	auth AuthPort
}

// Compile-time interface checks.
var (
	_ mono.Module                 = (*APIModule)(nil)
	_ mono.DependentModule        = (*APIModule)(nil)
	_ mono.HealthCheckableModule  = (*APIModule)(nil)
	_ types.ServiceConsumerModule = (*APIModule)(nil) // no mono.* alias in v0.1.1
)

// NewModule creates a new APIModule listening on DefaultAddr.
func NewModule() *APIModule {
	return NewModuleWithAddr(DefaultAddr)
}

// NewModuleWithAddr creates an APIModule listening on addr (tests use
// "127.0.0.1:0").
func NewModuleWithAddr(addr string) *APIModule {
	return &APIModule{addr: addr}
}

// Name returns the module name.
func (m *APIModule) Name() string {
	return "api"
}

// Dependencies returns the list of module dependencies.
func (m *APIModule) Dependencies() []string {
	return []string{"auth"}
}

// ConsumeServices declares every auth contract the handlers and the
// middleware call; the framework validates the list at startup.
func (m *APIModule) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		authcontract.Register.ToBase(),
		authcontract.Login.ToBase(),
		authcontract.RefreshToken.ToBase(),
		authcontract.ValidateToken.ToBase(),
		authcontract.GetUser.ToBase(),
	}
}

// SetDependencyServiceContainer wires the adapter behind AuthPort. The
// container is a read-only view of the auth module's services.
func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == "auth" {
		m.auth = authAdapter{container: container}
	}
}

// App returns the Fiber application (available after Start). Tests drive the
// routes with App().Test(req).
func (m *APIModule) App() *fiber.App {
	return m.app
}

// Start initializes the Fiber HTTP server.
func (m *APIModule) Start(_ context.Context) error {
	if m.auth == nil {
		return fmt.Errorf("auth dependency not set")
	}

	// Bind synchronously so a busy port fails Start instead of a goroutine.
	ln, err := net.Listen("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", m.addr, err)
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
		if err := m.app.Listener(ln); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Printf("[api] HTTP server started on %s", ln.Addr())
	return nil
}

// Stop shuts down the Fiber HTTP server.
func (m *APIModule) Stop(ctx context.Context) error {
	if m.app == nil {
		return nil
	}
	log.Println("[api] Shutting down HTTP server...")
	return m.app.ShutdownWithContext(ctx)
}

// Health returns the health status of the module.
func (m *APIModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"addr": m.addr,
		},
	}
}

// setupRoutes configures all API routes.
func (m *APIModule) setupRoutes() {
	handlers := NewHandlers(m.auth)

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
	protected.Use(AuthMiddleware(m.auth))
	protected.Get("/profile", handlers.Profile)
}

// customErrorHandler handles Fiber errors.
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	return c.Status(code).JSON(ErrorResponse{
		Error:   "server_error",
		Message: message,
	})
}
