// Package api is the driving adapter of the demo: a Fiber REST server that
// calls the core domain (task module) through the TaskPort it owns. It imports
// only task/contract, never the task module package.
package api

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	taskcontract "github.com/example/hexagonal-architecture/modules/task/contract"
)

// DefaultAddr is the address the HTTP server listens on.
const DefaultAddr = ":3000"

// APIModule is the driving adapter that exposes REST endpoints.
// It calls into the core domain (task module) via the TaskPort interface.
type APIModule struct {
	addr     string
	app      *fiber.App
	listener net.Listener
	tasks    TaskPort
}

var (
	_ mono.Module                 = (*APIModule)(nil)
	_ mono.DependentModule        = (*APIModule)(nil)
	_ mono.HealthCheckableModule  = (*APIModule)(nil)
	_ types.ServiceConsumerModule = (*APIModule)(nil) // no mono.* alias in v0.1.1
)

// NewModule creates the API module listening on DefaultAddr.
func NewModule() *APIModule {
	return NewModuleWithAddr(DefaultAddr)
}

// NewModuleWithAddr creates the API module listening on addr (tests use
// "127.0.0.1:0").
func NewModuleWithAddr(addr string) *APIModule {
	return &APIModule{addr: addr}
}

func (m *APIModule) Name() string {
	return "api"
}

func (m *APIModule) Dependencies() []string {
	return []string{"task"}
}

// ConsumeServices declares every task contract the HTTP handlers call.
func (m *APIModule) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		taskcontract.CreateTask.ToBase(),
		taskcontract.GetTask.ToBase(),
		taskcontract.UpdateTask.ToBase(),
		taskcontract.DeleteTask.ToBase(),
		taskcontract.ListTasks.ToBase(),
		taskcontract.CompleteTask.ToBase(),
	}
}

// SetDependencyServiceContainer wires the adapter behind TaskPort.
func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == "task" {
		m.tasks = taskAdapter{container: container}
	}
}

// App returns the Fiber application (available after Start). Tests drive the
// handlers with App().Test(req).
func (m *APIModule) App() *fiber.App {
	return m.app
}

func (m *APIModule) Start(_ context.Context) error {
	if m.tasks == nil {
		return fmt.Errorf("task dependency not set")
	}

	// Bind synchronously so a busy port fails Start instead of a goroutine.
	ln, err := net.Listen("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", m.addr, err)
	}
	m.listener = ln

	m.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler,
	})
	m.app.Use(recover.New())
	m.setupRoutes()

	go func() {
		if err := m.app.Listener(ln); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Printf("[api] HTTP server started on %s", ln.Addr())
	return nil
}

func (m *APIModule) Stop(ctx context.Context) error {
	if m.app == nil {
		return nil
	}
	log.Println("[api] Shutting down HTTP server...")
	return m.app.ShutdownWithContext(ctx)
}

func (m *APIModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.app != nil,
		Message: "operational",
		Details: map[string]any{
			"addr": m.addr,
		},
	}
}

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
