package api

import (
	"context"
	"fmt"
	"log"
	"net"

	productcontract "github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Module provides the HTTP API for the caching demo. It reaches the product
// module only through ProductPort, whose adapter calls the product contracts.
type Module struct {
	app      *fiber.App
	handlers *Handlers
	products ProductPort
	port     int
	addr     string
}

// Compile-time interface checks.
var (
	_ mono.Module                 = (*Module)(nil)
	_ mono.DependentModule        = (*Module)(nil)
	_ types.ServiceConsumerModule = (*Module)(nil) // no mono.* alias in v0.1.1
)

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

// Dependencies declares the product module: it starts first, and its
// container is handed to SetDependencyServiceContainer.
func (m *Module) Dependencies() []string {
	return []string{productcontract.ModuleName}
}

// SetDependencyServiceContainer wires the adapter behind ProductPort. The
// container is a read-only view: only look services up on it.
func (m *Module) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == productcontract.ModuleName {
		m.products = productAdapter{container: container}
	}
}

// ConsumeServices declares every product contract the api module calls. The
// framework validates the list at start: product must be a dependency, must
// have registered each service, and the service kinds must match.
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		productcontract.ListProducts.ToBase(),
		productcontract.GetProduct.ToBase(),
		productcontract.CreateProduct.ToBase(),
		productcontract.UpdateProduct.ToBase(),
		productcontract.DeleteProduct.ToBase(),
	}
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

	if m.products == nil {
		return fmt.Errorf("product dependency not set - ensure the product module is registered")
	}
	m.handlers = NewHandlers(m.products)

	// Setup routes
	m.setupRoutes()

	// Bind synchronously so a busy port fails Start instead of a goroutine.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", m.port, err)
	}
	m.addr = ln.Addr().String()

	go func() {
		log.Printf("[api] Starting HTTP server on %s", m.addr)
		if err := m.app.Listener(ln); err != nil {
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

// Addr returns the address the HTTP server listens on (empty before Start).
func (m *Module) Addr() string {
	return m.addr
}
