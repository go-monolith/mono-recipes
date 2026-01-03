package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Handlers provides HTTP handlers for the API endpoints.
type Handlers struct{}

// NewHandlers creates a new handlers instance.
func NewHandlers() *Handlers {
	return &Handlers{}
}

// PublicEndpoint handles requests to the public endpoint.
// This endpoint is rate limited by IP address (100 req/min).
func (h *Handlers) PublicEndpoint(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message":   "Welcome to the public API!",
		"endpoint":  "/api/v1/public",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"client_ip": c.IP(),
		"rate_limit": fiber.Map{
			"type":  "ip-based",
			"limit": "100 requests per minute",
		},
	})
}

// PremiumEndpoint handles requests to the premium endpoint.
// This endpoint is rate limited by API key (1000 req/min for authenticated users).
func (h *Handlers) PremiumEndpoint(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		apiKey = "anonymous"
	}

	return c.JSON(fiber.Map{
		"message":   "Welcome to the premium API!",
		"endpoint":  "/api/v1/premium",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"api_key":   maskAPIKey(apiKey),
		"rate_limit": fiber.Map{
			"type":  "api-key-based",
			"limit": "1000 requests per minute",
		},
	})
}

// HealthEndpoint returns the health status of the service.
func (h *Handlers) HealthEndpoint(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"service":   "rate-limiting-demo",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// StatsEndpoint returns rate limiting statistics.
// This is useful for monitoring and debugging.
func (h *Handlers) StatsEndpoint(statsFunc func(c *fiber.Ctx) (map[string]interface{}, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := statsFunc(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to retrieve statistics",
				"message": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"stats":     stats,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// maskAPIKey masks the API key for display, showing only first 4 and last 4 characters.
func maskAPIKey(key string) string {
	if key == "anonymous" || len(key) <= 8 {
		return key
	}
	return key[:4] + "****" + key[len(key)-4:]
}
