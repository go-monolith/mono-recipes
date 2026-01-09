package httpserver

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/example/url-shortener-demo/modules/analytics"
	"github.com/example/url-shortener-demo/modules/shortener"
)

// Handlers contains HTTP request handlers for URL shortener operations.
type Handlers struct {
	shortener shortener.ShortenerAdapterPort
	analytics analytics.AnalyticsAdapterPort
}

// NewHandlers creates a new handlers instance.
func NewHandlers(shortener shortener.ShortenerAdapterPort, analytics analytics.AnalyticsAdapterPort) *Handlers {
	return &Handlers{
		shortener: shortener,
		analytics: analytics,
	}
}

// ShortenURL handles URL shortening requests (POST /api/v1/shorten).
func (h *Handlers) ShortenURL(c *fiber.Ctx) error {
	var req shortener.ShortenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL is required",
		})
	}

	result, err := h.shortener.ShortenURL(c.Context(), req)
	if err != nil {
		if errors.Is(err, shortener.ErrInvalidURL) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid URL",
				"details": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to shorten URL",
			"details": err.Error(),
		})
	}

	// Event publishing is now handled internally by the shortener service
	return c.Status(fiber.StatusCreated).JSON(result)
}

// Redirect handles URL redirect requests (GET /:shortCode).
func (h *Handlers) Redirect(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	if shortCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Short code is required",
		})
	}

	originalURL, err := h.shortener.ResolveURL(c.Context(), shortCode, c.Get("User-Agent"), c.IP())
	if err != nil {
		if errors.Is(err, shortener.ErrURLNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "URL not found",
			})
		}
		if errors.Is(err, shortener.ErrInvalidShortCode) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid short code format",
			})
		}
		if errors.Is(err, shortener.ErrURLExpired) {
			return c.Status(fiber.StatusGone).JSON(fiber.Map{
				"error": "URL has expired",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to resolve URL",
			"details": err.Error(),
		})
	}

	// Event publishing is now handled internally by the shortener service
	return c.Redirect(originalURL, fiber.StatusTemporaryRedirect)
}

// GetStats handles statistics requests (GET /api/v1/stats/:shortCode).
func (h *Handlers) GetStats(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	if shortCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Short code is required",
		})
	}

	stats, err := h.shortener.GetStats(c.Context(), shortCode)
	if err != nil {
		if errors.Is(err, shortener.ErrURLNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "URL not found",
			})
		}
		if errors.Is(err, shortener.ErrInvalidShortCode) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid short code format",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get stats",
			"details": err.Error(),
		})
	}

	return c.JSON(stats)
}

// ListURLs handles URL listing requests (GET /api/v1/urls).
func (h *Handlers) ListURLs(c *fiber.Ctx) error {
	urls, err := h.shortener.ListURLs(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to list URLs",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"urls":  urls,
		"total": len(urls),
	})
}

// DeleteURL handles URL deletion requests (DELETE /api/v1/urls/:shortCode).
func (h *Handlers) DeleteURL(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	if shortCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Short code is required",
		})
	}

	err := h.shortener.DeleteURL(c.Context(), shortCode)
	if err != nil {
		if errors.Is(err, shortener.ErrURLNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "URL not found",
			})
		}
		if errors.Is(err, shortener.ErrInvalidShortCode) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid short code format",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to delete URL",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":    "URL deleted successfully",
		"short_code": shortCode,
	})
}

// GetAnalytics handles analytics summary requests (GET /api/v1/analytics).
func (h *Handlers) GetAnalytics(c *fiber.Ctx) error {
	summary, err := h.analytics.GetSummary(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get analytics summary",
			"details": err.Error(),
		})
	}
	return c.JSON(summary)
}

// GetAnalyticsLogs handles recent access logs requests (GET /api/v1/analytics/logs).
func (h *Handlers) GetAnalyticsLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	if limit > 1000 {
		limit = 1000
	}

	logs, err := h.analytics.GetRecentLogs(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get analytics logs",
			"details": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"logs":  logs,
		"total": len(logs),
	})
}

// HealthCheck handles health check requests (GET /health).
func (h *Handlers) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "url-shortener-demo",
	})
}
