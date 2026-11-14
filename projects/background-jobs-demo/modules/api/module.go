// Package api provides REST API handlers for job management. It holds no job
// state: it creates and queries jobs on the worker module (their owner)
// through worker/contract, behind the JobsPort it owns, and imports nothing
// else of the worker module.
package api

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/fiber/v2"

	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// Module provides the REST API as a mono module.
type Module struct {
	app     *fiber.App
	handler *Handler
	service *Service
	jobs    JobsPort
	port    int
	addr    string
}

// Compile-time interface checks.
var (
	_ mono.Module                 = (*Module)(nil)
	_ mono.DependentModule        = (*Module)(nil)
	_ types.ServiceConsumerModule = (*Module)(nil) // no mono.* alias in v0.1.1
)

// NewModule creates a new API module listening on :port. port 0 picks a free
// port (used by tests).
func NewModule(port int) *Module {
	return &Module{port: port}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "api"
}

// Dependencies declares that this module depends on the worker module.
func (m *Module) Dependencies() []string {
	return []string{"worker"}
}

// ConsumeServices declares every worker contract this module calls; the
// framework validates them at startup.
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		workercontract.CreateJob.ToBase(),
		workercontract.GetJob.ToBase(),
		workercontract.ListJobs.ToBase(),
	}
}

// SetDependencyServiceContainer wires the adapter behind JobsPort. The
// container is a read-only view of the worker module's services.
func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
	if module == "worker" {
		m.jobs = workerAdapter{container: container}
	}
}

// Start initializes and starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	if m.jobs == nil {
		return fmt.Errorf("worker dependency not set")
	}

	// Bind synchronously so a busy port fails Start instead of a goroutine.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.port))
	if err != nil {
		return fmt.Errorf("listen on :%d: %w", m.port, err)
	}
	m.addr = ln.Addr().String()

	// Create Fiber app
	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "internal_error",
				"message": "An unexpected error occurred",
			})
		},
	})

	// Create service and handler
	m.service = NewService(m.jobs)
	m.handler = NewHandler(m.service)

	// Register routes
	m.handler.RegisterRoutes(m.app)

	// Health endpoint
	m.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "background-jobs-demo",
		})
	})

	// Start HTTP server in background
	go func() {
		log.Printf("[api] Starting HTTP server on %s", m.addr)
		if err := m.app.Listener(ln); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Println("[api] API module started")
	return nil
}

// Stop stops the HTTP server gracefully.
func (m *Module) Stop(ctx context.Context) error {
	if m.app != nil {
		if err := m.app.ShutdownWithContext(ctx); err != nil {
			return err
		}
	}
	log.Println("[api] Module stopped")
	return nil
}

// GetApp returns the Fiber app instance.
func (m *Module) GetApp() *fiber.App {
	return m.app
}
