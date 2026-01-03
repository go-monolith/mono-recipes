package api

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/example/url-shortener-demo/modules/shortener"
	"github.com/gofiber/fiber/v2"
)

const maxURLLength = 2048

// isPrivateOrReservedIP checks if an IP address is private, loopback, or reserved.
// This is used to prevent SSRF attacks by blocking URLs pointing to internal networks.
func isPrivateOrReservedIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// Check for loopback, private, link-local, and other reserved ranges
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// isBlockedHost checks if the URL hostname is blocked (SSRF protection).
// Blocks localhost, private IPs, and cloud metadata endpoints.
func isBlockedHost(hostname string) bool {
	// Block common localhost aliases
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
		return true
	}

	// Block cloud metadata endpoints
	if lowerHost == "169.254.169.254" || lowerHost == "metadata.google.internal" {
		return true
	}

	// Parse IP and check if it's private/reserved
	ip := net.ParseIP(hostname)
	if ip != nil && isPrivateOrReservedIP(ip) {
		return true
	}

	return false
}

// setupRoutes configures all HTTP routes.
func (m *APIModule) setupRoutes() {
	// Health check endpoint
	m.app.Get("/health", m.healthHandler)

	// API v1 routes
	api := m.app.Group("/api/v1")

	// Shorten endpoint
	api.Post("/shorten", m.shortenURL)

	// Stats endpoint
	api.Get("/stats/:shortCode", m.getStats)

	// Redirect endpoint (root level for short URLs)
	m.app.Get("/:shortCode", m.redirectURL)
}

// healthHandler handles GET /health.
func (m *APIModule) healthHandler(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status: "healthy",
		Details: map[string]any{
			"module": "api",
			"port":   m.port,
		},
	})
}

// shortenURL handles POST /api/v1/shorten.
func (m *APIModule) shortenURL(c *fiber.Ctx) error {
	var req ShortenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate URL
	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "URL is required",
		})
	}

	if len(req.URL) > maxURLLength {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: fmt.Sprintf("URL exceeds maximum length of %d characters", maxURLLength),
		})
	}

	// Validate URL format
	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid URL format. Must be a valid HTTP or HTTPS URL",
		})
	}

	// SSRF protection: block private/internal URLs
	if isBlockedHost(parsedURL.Hostname()) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "URL points to a blocked host (private or internal network)",
		})
	}

	// Validate custom code if provided
	if req.CustomCode != "" && !shortener.IsValidShortCode(req.CustomCode) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid custom code. Must be alphanumeric and max 20 characters",
		})
	}

	// Shorten URL via adapter
	resp, err := m.shortenerAdapter.ShortenURL(c.UserContext(), req.URL, req.CustomCode, req.TTLSeconds)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "already in use") {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error:   "conflict",
				Message: "Custom code already in use",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "shorten_failed",
			Message: "Failed to shorten URL",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(ShortenResponse{
		ID:          resp.ID,
		ShortCode:   resp.ShortCode,
		ShortURL:    resp.ShortURL,
		OriginalURL: resp.OriginalURL,
		CreatedAt:   resp.CreatedAt,
		ExpiresAt:   resp.ExpiresAt,
	})
}

// redirectURL handles GET /:shortCode (redirect to original URL).
func (m *APIModule) redirectURL(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	// Skip if it looks like an API path
	if shortCode == "api" || shortCode == "health" {
		return c.Next()
	}

	if !shortener.IsValidShortCode(shortCode) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid short code format",
		})
	}

	// Resolve URL
	resp, err := m.shortenerAdapter.ResolveURL(c.UserContext(), shortCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Short URL not found",
		})
	}

	// Record access asynchronously (non-blocking)
	// Use background context since the request context may be cancelled
	// before the async operation completes
	go func() {
		userAgent := c.Get("User-Agent")
		referer := c.Get("Referer")
		ipAddress := c.IP()
		_ = m.shortenerAdapter.RecordAccess(context.Background(), shortCode, userAgent, referer, ipAddress)
	}()

	// Redirect to original URL using 302 (Found) instead of 301 (Moved Permanently)
	// to allow URL reassignment without browser caching issues
	return c.Redirect(resp.OriginalURL, fiber.StatusFound)
}

// getStats handles GET /api/v1/stats/:shortCode.
func (m *APIModule) getStats(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")

	if !shortener.IsValidShortCode(shortCode) {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid short code format",
		})
	}

	stats, err := m.shortenerAdapter.GetStats(c.UserContext(), shortCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Short URL not found",
		})
	}

	return c.JSON(StatsResponse{
		ShortCode:   stats.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", m.baseURL, stats.ShortCode),
		OriginalURL: stats.OriginalURL,
		AccessCount: stats.AccessCount,
		CreatedAt:   stats.CreatedAt,
		LastAccess:  stats.LastAccess,
	})
}
