package api

import (
	"context"
	"fmt"
	"log"

	ratelimitmod "github.com/example/rate-limiting-demo/modules/ratelimit"
	"github.com/go-monolith/mono"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Module provides the HTTP API for the rate limiting demo.
type Module struct {
	app             *fiber.App
	handlers        *Handlers
	rateLimitModule *ratelimitmod.Module
	port            int
}

// NewModule creates a new API module.
func NewModule(port int) *Module {
	return &Module{
		port:     port,
		handlers: NewHandlers(),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "api"
}

// SetRateLimitModule sets the rate limiting module dependency.
func (m *Module) SetRateLimitModule(rlm *ratelimitmod.Module) {
	m.rateLimitModule = rlm
}

// Init initializes the Fiber app and configures routes.
func (m *Module) Init(_ mono.ServiceContainer) error {
	m.app = fiber.New(fiber.Config{
		AppName:               "Rate Limiting Demo",
		DisableStartupMessage: true,
		ErrorHandler:          m.errorHandler,
	})

	// Global middleware
	m.app.Use(recover.New())
	m.app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	m.app.Use(cors.New())

	// Setup routes
	m.setupRoutes()

	return nil
}

// setupRoutes configures all HTTP routes.
func (m *Module) setupRoutes() {
	// Health check (no rate limiting)
	m.app.Get("/health", m.handlers.HealthEndpoint)

	// API v1 routes
	api := m.app.Group("/api/v1")

	if m.rateLimitModule != nil {
		middleware := m.rateLimitModule.GetMiddleware()

		// Public endpoint with IP-based rate limiting (100 req/min)
		api.Get("/public", middleware.IPRateLimit(), m.handlers.PublicEndpoint)

		// Premium endpoint with API key-based rate limiting (1000 req/min)
		api.Get("/premium", middleware.APIKeyRateLimit(), m.handlers.PremiumEndpoint)

		// Stats endpoint (no rate limiting, for monitoring)
		api.Get("/stats", m.handlers.StatsEndpoint(func(c *fiber.Ctx) (map[string]interface{}, error) {
			ip := c.IP()
			apiKey := c.Get("X-API-Key")

			ctx := c.Context()
			stats := make(map[string]interface{})

			// Get IP-based stats
			ipStats, err := middleware.GetIPLimiter().GetStats(ctx, ip)
			if err == nil {
				stats["ip_rate_limit"] = ipStats
			}

			// Get API key stats if provided
			if apiKey != "" {
				apiKeyStats, err := middleware.GetUserLimiter().GetStats(ctx, "apikey:"+apiKey)
				if err == nil {
					stats["api_key_rate_limit"] = apiKeyStats
				}
			}

			return stats, nil
		}))
	} else {
		// Fallback routes without rate limiting
		api.Get("/public", m.handlers.PublicEndpoint)
		api.Get("/premium", m.handlers.PremiumEndpoint)
	}
}

// Start starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	go func() {
		addr := fmt.Sprintf(":%d", m.port)
		log.Printf("[api] Starting HTTP server on %s", addr)
		if err := m.app.Listen(addr); err != nil {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()
	return nil
}

// Stop stops the HTTP server gracefully.
func (m *Module) Stop(_ context.Context) error {
	if m.app != nil {
		log.Println("[api] Shutting down HTTP server...")
		return m.app.Shutdown()
	}
	return nil
}

// errorHandler handles errors from Fiber routes.
func (m *Module) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"code":    code,
		"path":    c.Path(),
		"method":  c.Method(),
	})
}

// GetApp returns the Fiber app (for testing).
func (m *Module) GetApp() *fiber.App {
	return m.app
}
