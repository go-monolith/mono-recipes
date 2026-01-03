package ratelimit

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/rate-limiting-demo/domain/ratelimit"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// setupTestApp creates a Fiber app with rate limiting middleware for testing.
func setupTestApp(t *testing.T) (*fiber.App, *Middleware, func()) {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := client.Ping(t.Context()).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	// Use short windows for testing
	config := ratelimit.MiddlewareConfig{
		IPConfig: ratelimit.Config{
			RequestsPerWindow: 3,
			WindowSize:        time.Minute,
		},
		UserConfig: ratelimit.Config{
			RequestsPerWindow: 5,
			WindowSize:        time.Minute,
		},
		GlobalConfig: ratelimit.Config{
			RequestsPerWindow: 100,
			WindowSize:        time.Minute,
		},
		KeyPrefix: "test:middleware:",
	}

	middleware := NewMiddleware(client, config)

	app := fiber.New()

	// Cleanup function
	cleanup := func() {
		app.Shutdown()
		// Clean up test keys
		keys, _ := client.Keys(t.Context(), "test:middleware:*").Result()
		if len(keys) > 0 {
			client.Del(t.Context(), keys...)
		}
		client.Close()
	}

	return app, middleware, cleanup
}

// TestMiddleware_IPRateLimit tests IP-based rate limiting.
func TestMiddleware_IPRateLimit(t *testing.T) {
	app, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	app.Get("/test", middleware.IPRateLimit(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.100")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}

		// Check rate limit headers
		limit := resp.Header.Get("X-RateLimit-Limit")
		if limit != "3" {
			t.Errorf("Expected X-RateLimit-Limit=3, got %s", limit)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}

	// Check Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Error("Expected Retry-After header")
	}
}

// TestMiddleware_APIKeyRateLimit tests API key-based rate limiting.
func TestMiddleware_APIKeyRateLimit(t *testing.T) {
	app, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	app.Get("/premium", middleware.APIKeyRateLimit(), func(c *fiber.Ctx) error {
		return c.SendString("Premium")
	})

	apiKey := "test-api-key-12345"

	// First 5 requests should succeed (user config has limit of 5)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/premium", nil)
		req.Header.Set("X-API-Key", apiKey)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("GET", "/premium", nil)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}

	// Different API key should still work
	req = httptest.NewRequest("GET", "/premium", nil)
	req.Header.Set("X-API-Key", "different-api-key")

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Different API key should not be rate limited, got %d", resp.StatusCode)
	}
}

// TestMiddleware_APIKeyFallbackToIP tests that missing API key falls back to IP.
func TestMiddleware_APIKeyFallbackToIP(t *testing.T) {
	app, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	app.Get("/premium", middleware.APIKeyRateLimit(), func(c *fiber.Ctx) error {
		return c.SendString("Premium")
	})

	// Request without API key should use IP rate limiting (limit of 3)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/premium", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 4th request without API key should hit IP limit
	req := httptest.NewRequest("GET", "/premium", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}
}

// TestMiddleware_GlobalRateLimit tests global rate limiting.
func TestMiddleware_GlobalRateLimit(t *testing.T) {
	app, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	app.Get("/global", middleware.GlobalRateLimit(), func(c *fiber.Ctx) error {
		return c.SendString("Global")
	})

	// Global limit is 100, so first request should succeed
	req := httptest.NewRequest("GET", "/global", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check header shows correct limit
	limit := resp.Header.Get("X-RateLimit-Limit")
	if limit != "100" {
		t.Errorf("Expected X-RateLimit-Limit=100, got %s", limit)
	}
}

// TestMiddleware_RateLimitResponse tests the 429 response format.
func TestMiddleware_RateLimitResponse(t *testing.T) {
	app, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	app.Get("/test", middleware.IPRateLimit(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Exhaust the limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		app.Test(req)
	}

	// Get rate limited response
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}

	// Check response body
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if bodyStr == "" {
		t.Error("Expected non-empty response body")
	}

	// Should contain error information
	if !contains(bodyStr, "Too Many Requests") {
		t.Errorf("Response should contain 'Too Many Requests', got: %s", bodyStr)
	}
}

// TestMiddleware_CustomRateLimit tests custom rate limiting.
func TestMiddleware_CustomRateLimit(t *testing.T) {
	app, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	// Custom rate limit of 2 requests per minute
	customConfig := ratelimit.Config{
		RequestsPerWindow: 2,
		WindowSize:        time.Minute,
	}

	// Extract key from custom header
	keyExtractor := func(c *fiber.Ctx) string {
		return c.Get("X-Custom-ID")
	}

	app.Get("/custom", middleware.CustomRateLimit(customConfig, keyExtractor), func(c *fiber.Ctx) error {
		return c.SendString("Custom")
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/custom", nil)
		req.Header.Set("X-Custom-ID", "custom-user-1")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/custom", nil)
	req.Header.Set("X-Custom-ID", "custom-user-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}
}

// TestMiddleware_GetLimiters tests limiter accessors.
func TestMiddleware_GetLimiters(t *testing.T) {
	_, middleware, cleanup := setupTestApp(t)
	defer cleanup()

	if middleware.GetIPLimiter() == nil {
		t.Error("GetIPLimiter should not return nil")
	}

	if middleware.GetUserLimiter() == nil {
		t.Error("GetUserLimiter should not return nil")
	}

	if middleware.GetGlobalLimiter() == nil {
		t.Error("GetGlobalLimiter should not return nil")
	}
}

// contains checks if substr is in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
