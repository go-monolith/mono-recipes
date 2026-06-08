package api

import (
	"context"
	"log"

	"github.com/example/cron-scheduler-demo/stats"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
)

// Module serves a small HTTP API to observe the cron activity recorded by the
// scheduler module. It reads the same shared stats.Store the scheduler writes,
// so /stats reflects live cron occurrences.
type Module struct {
	store *stats.Store
	addr  string
	app   *fiber.App
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.HealthCheckableModule = (*Module)(nil)
)

// NewModule creates the API module bound to addr (e.g. ":8080").
func NewModule(store *stats.Store, addr string) *Module {
	return &Module{store: store, addr: addr}
}

// Name identifies the module.
func (m *Module) Name() string { return "api" }

// Start brings up the Fiber HTTP server in the background.
func (m *Module) Start(_ context.Context) error {
	m.app = fiber.New(fiber.Config{DisableStartupMessage: true})

	m.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	// GET /stats returns per-schedule tick counts as JSON, e.g.
	// {"heartbeat":{"ticks":12,"last_tick":"..."}, "metrics":{...}}
	m.app.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(m.store.Snapshot())
	})

	go func() {
		if err := m.app.Listen(m.addr); err != nil {
			log.Printf("[api] server error: %v", err)
		}
	}()

	log.Printf("[api] listening on %s (GET /health, GET /stats)", m.addr)
	return nil
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
