package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	schedulercontract "github.com/example/cron-scheduler-demo/modules/scheduler/contract"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/gofiber/fiber/v2"
)

// requestTimeout bounds each GetStats call to the scheduler module; the
// deadline is propagated to the scheduler's handler.
const requestTimeout = 3 * time.Second

// Module serves a small HTTP API to observe the cron activity recorded by the
// scheduler module. It reads the stats through StatsPort, whose adapter calls
// the scheduler's GetStats contract, so /stats reflects live cron occurrences.
type Module struct {
	stats StatsPort
	addr  string
	app   *fiber.App
	ln    net.Listener
}

// Compile-time interface checks.
var (
	_ mono.Module                 = (*Module)(nil)
	_ mono.DependentModule        = (*Module)(nil)
	_ mono.HealthCheckableModule  = (*Module)(nil)
	_ types.ServiceConsumerModule = (*Module)(nil) // no mono.* alias in v0.1.1
)

// NewModule creates the API module bound to addr (e.g. ":8080").
func NewModule(addr string) *Module {
	return &Module{addr: addr}
}

// Name identifies the module.
func (m *Module) Name() string { return "api" }

// Dependencies declares the scheduler module: it starts first, and its
// container is handed to SetDependencyServiceContainer.
func (m *Module) Dependencies() []string { return []string{schedulercontract.ModuleName} }

// SetDependencyServiceContainer wires the adapter behind StatsPort. The
// container is a read-only view: only look services up on it.
func (m *Module) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == schedulercontract.ModuleName {
		m.stats = schedulerAdapter{container: container}
	}
}

// ConsumeServices declares every contract the api module calls; the framework
// validates it at start (scheduler is a dependency and registered get-stats).
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{schedulercontract.GetStats.ToBase()}
}

// Start brings up the Fiber HTTP server in the background.
func (m *Module) Start(_ context.Context) error {
	if m.stats == nil {
		return fmt.Errorf("scheduler dependency not set - ensure the scheduler module is registered")
	}

	m.app = fiber.New(fiber.Config{DisableStartupMessage: true})

	m.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	// GET /stats returns per-schedule tick counts as JSON, e.g.
	// {"heartbeat":{"ticks":12,"last_tick":"..."}, "metrics":{...}}
	m.app.Get("/stats", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), requestTimeout)
		defer cancel()
		snapshot, err := m.stats.GetStats(ctx)
		if err != nil {
			log.Printf("[api] get stats: %v", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "stats unavailable"})
		}
		return c.JSON(snapshot)
	})

	// Bind synchronously so a busy port fails Start instead of a goroutine.
	ln, err := net.Listen("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", m.addr, err)
	}
	m.ln = ln

	go func() {
		if err := m.app.Listener(ln); err != nil {
			log.Printf("[api] server error: %v", err)
		}
	}()

	log.Printf("[api] listening on %s (GET /health, GET /stats)", ln.Addr())
	return nil
}

// Addr returns the address the HTTP server listens on: the bound address once
// started (useful with port 0), the configured one before.
func (m *Module) Addr() string {
	if m.ln != nil {
		return m.ln.Addr().String()
	}
	return m.addr
}

// Stop gracefully shuts down the HTTP server, draining in-flight requests.
func (m *Module) Stop(ctx context.Context) error {
	if m.app == nil {
		return nil
	}
	return m.app.ShutdownWithContext(ctx)
}

// Health reports the module status.
func (m *Module) Health(_ context.Context) mono.HealthStatus {
	if m.app == nil {
		return mono.HealthStatus{Healthy: false, Message: "HTTP server not initialized"}
	}
	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{"addr": m.addr},
	}
}
