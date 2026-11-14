package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
)

// Config configures the HTTP server module. Zero values use the defaults.
type Config struct {
	// Addr is the listen address (default ":3000"). Use "127.0.0.1:0" in tests.
	Addr string
	// SlowDelay is how long GET /slow takes (default 5s).
	SlowDelay time.Duration
}

// HttpServerModule implements the mono Module interface for an HTTP server.
//
// Graceful shutdown semantics: Stop stops accepting connections and waits for
// in-flight requests to finish, bounded by the deadline of the context the
// framework passes to Stop (the app's shutdown timeout). Requests still running
// when that deadline passes are cancelled through requestCtx.
type HttpServerModule struct {
	addr      string
	slowDelay time.Duration

	app *fiber.App
	ln  net.Listener

	// requestCtx is the parent of every in-flight request's work. It is
	// cancelled only when the shutdown deadline is exceeded (or after Stop).
	// fiber's c.Context().Done() is not used: fasthttp closes it as soon as
	// shutdown starts, which would abort requests instead of draining them.
	requestCtx    context.Context
	cancelRequest context.CancelFunc
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*HttpServerModule)(nil)
	_ mono.HealthCheckableModule = (*HttpServerModule)(nil)
)

// NewModule creates the HTTP server module.
func NewModule(cfg Config) *HttpServerModule {
	if cfg.Addr == "" {
		cfg.Addr = ":3000"
	}
	if cfg.SlowDelay <= 0 {
		cfg.SlowDelay = 5 * time.Second
	}
	return &HttpServerModule{addr: cfg.Addr, slowDelay: cfg.SlowDelay}
}

// Name returns the module name.
func (m *HttpServerModule) Name() string {
	return "http-server"
}

// Addr returns the address the server listens on: the bound address once
// started (useful with port 0), the configured one before.
func (m *HttpServerModule) Addr() string {
	if m.ln != nil {
		return m.ln.Addr().String()
	}
	return m.addr
}

// Health performs a health check on the HTTP server module.
func (m *HttpServerModule) Health(_ context.Context) mono.HealthStatus {
	if m.app == nil || m.ln == nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: "HTTP server not initialized",
		}
	}
	port := 0
	if tcp, ok := m.ln.Addr().(*net.TCPAddr); ok {
		port = tcp.Port
	}
	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"port": port,
		},
	}
}

// Start initializes and starts the HTTP server.
func (m *HttpServerModule) Start(_ context.Context) error {
	m.requestCtx, m.cancelRequest = context.WithCancel(context.Background())
	m.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	m.registerRoutes()

	return m.startServer()
}

// registerRoutes sets up all HTTP route handlers.
func (m *HttpServerModule) registerRoutes() {
	m.app.Get("/", m.handleRoot)
	m.app.Get("/health", m.handleHealth)
	m.app.Get("/slow", m.handleSlow)
}

func (m *HttpServerModule) handleRoot(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Hello from graceful-shutdown-demo!",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (m *HttpServerModule) handleHealth(c *fiber.Ctx) error {
	return c.JSON(m.Health(c.UserContext()))
}

// handleSlow demonstrates in-flight request handling during shutdown: a
// request that started before shutdown completes, unless the shutdown
// deadline passes first.
func (m *HttpServerModule) handleSlow(c *fiber.Ctx) error {
	log.Println("Slow request started...")

	select {
	case <-time.After(m.slowDelay):
		log.Println("Slow request completed!")
		return c.JSON(fiber.Map{
			"message": fmt.Sprintf("This request took %v seconds to complete", m.slowDelay.Seconds()),
			"time":    time.Now().Format(time.RFC3339),
		})
	case <-m.requestCtx.Done():
		log.Println("Slow request cancelled: shutdown deadline exceeded")
		return fiber.NewError(fiber.StatusServiceUnavailable, "server shutting down")
	}
}

// startServer binds the listener synchronously (so a busy port fails Start)
// and serves on it in a goroutine.
func (m *HttpServerModule) startServer() error {
	ln, err := net.Listen("tcp", m.addr)
	if err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	m.ln = ln

	go func() {
		if err := m.app.Listener(ln); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("HTTP server started on %s", ln.Addr())
	return nil
}

// Stop gracefully shuts down the HTTP server: it stops accepting connections,
// waits for in-flight requests until ctx's deadline, then cancels the requests
// still running. It returns within ctx's deadline.
func (m *HttpServerModule) Stop(ctx context.Context) error {
	if m.app == nil {
		return nil
	}

	log.Println("Shutting down HTTP server...")

	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			m.cancelRequest() // deadline exceeded: abort what is still running
		case <-stopped:
		}
	}()

	err := m.app.ShutdownWithContext(ctx)
	m.cancelRequest()
	// Close the listener too, in case Stop runs before the serve goroutine
	// registered it with fasthttp (then ShutdownWithContext has nothing to close).
	_ = m.ln.Close()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			log.Println("HTTP server shutdown deadline exceeded; in-flight requests cancelled")
		}
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	log.Println("HTTP server stopped gracefully")
	return nil
}
