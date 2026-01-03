package ratelimit

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/example/rate-limiting-demo/domain/ratelimit"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// Middleware provides rate limiting middleware for Fiber.
type Middleware struct {
	ipLimiter     *SlidingWindowLimiter
	userLimiter   *SlidingWindowLimiter
	globalLimiter *SlidingWindowLimiter
	config        ratelimit.MiddlewareConfig
}

// NewMiddleware creates a new rate limiting middleware.
func NewMiddleware(client *redis.Client, config ratelimit.MiddlewareConfig) *Middleware {
	return &Middleware{
		ipLimiter:     NewSlidingWindowLimiter(client, config.IPConfig, config.KeyPrefix+"ip:"),
		userLimiter:   NewSlidingWindowLimiter(client, config.UserConfig, config.KeyPrefix+"user:"),
		globalLimiter: NewSlidingWindowLimiter(client, config.GlobalConfig, config.KeyPrefix+"global:"),
		config:        config,
	}
}

// IPRateLimit returns middleware that limits requests by client IP.
func (m *Middleware) IPRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if ip == "" {
			// Fail closed: reject requests when IP cannot be determined
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Forbidden",
				"message": "Unable to determine client IP address",
			})
		}

		result, err := m.ipLimiter.Allow(c.Context(), ip)
		if err != nil {
			// On error, allow the request but log it
			c.Set("X-RateLimit-Error", err.Error())
			return c.Next()
		}

		setRateLimitHeaders(c, result, m.config.IPConfig.RequestsPerWindow)

		if !result.Allowed {
			return sendRateLimitExceeded(c, result)
		}

		return c.Next()
	}
}

// UserRateLimit returns middleware that limits requests by user ID.
// It expects the user ID to be set in c.Locals("user_id").
func (m *Middleware) UserRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			// No user ID, fall back to IP-based limiting
			return m.IPRateLimit()(c)
		}

		result, err := m.userLimiter.Allow(c.Context(), userID)
		if err != nil {
			c.Set("X-RateLimit-Error", err.Error())
			return c.Next()
		}

		setRateLimitHeaders(c, result, m.config.UserConfig.RequestsPerWindow)

		if !result.Allowed {
			return sendRateLimitExceeded(c, result)
		}

		return c.Next()
	}
}

// GlobalRateLimit returns middleware that applies a global rate limit.
// This is useful as a safety net for all requests.
func (m *Middleware) GlobalRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := m.globalLimiter.Allow(c.Context(), "all")
		if err != nil {
			c.Set("X-RateLimit-Error", err.Error())
			return c.Next()
		}

		setRateLimitHeaders(c, result, m.config.GlobalConfig.RequestsPerWindow)

		if !result.Allowed {
			return sendRateLimitExceeded(c, result)
		}

		return c.Next()
	}
}

// APIKeyRateLimit returns middleware that limits requests by API key.
// It expects the API key to be set in the X-API-Key header or Authorization header.
func (m *Middleware) APIKeyRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			apiKey = c.Get("Authorization")
		}

		if apiKey == "" {
			// No API key, fall back to IP-based limiting
			return m.IPRateLimit()(c)
		}

		// Validate API key format to prevent Redis key injection
		if !isValidAPIKey(apiKey) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Bad Request",
				"message": "Invalid API key format",
			})
		}

		result, err := m.userLimiter.Allow(c.Context(), "apikey:"+apiKey)
		if err != nil {
			c.Set("X-RateLimit-Error", err.Error())
			return c.Next()
		}

		setRateLimitHeaders(c, result, m.config.UserConfig.RequestsPerWindow)

		if !result.Allowed {
			return sendRateLimitExceeded(c, result)
		}

		return c.Next()
	}
}

// CustomRateLimit returns middleware with a custom rate limit configuration.
func (m *Middleware) CustomRateLimit(config ratelimit.Config, keyExtractor func(*fiber.Ctx) string) fiber.Handler {
	limiter := NewSlidingWindowLimiter(m.ipLimiter.client, config, m.config.KeyPrefix+"custom:")

	return func(c *fiber.Ctx) error {
		key := keyExtractor(c)
		if key == "" {
			key = c.IP()
		}

		result, err := limiter.Allow(c.Context(), key)
		if err != nil {
			c.Set("X-RateLimit-Error", err.Error())
			return c.Next()
		}

		setRateLimitHeaders(c, result, config.RequestsPerWindow)

		if !result.Allowed {
			return sendRateLimitExceeded(c, result)
		}

		return c.Next()
	}
}

// setRateLimitHeaders sets standard rate limit headers on the response.
func setRateLimitHeaders(c *fiber.Ctx, result *ratelimit.Result, limit int) {
	c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	c.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
}

// sendRateLimitExceeded sends a 429 Too Many Requests response.
func sendRateLimitExceeded(c *fiber.Ctx, result *ratelimit.Result) error {
	retryAfter := int(result.RetryAfter.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}

	c.Set("Retry-After", strconv.Itoa(retryAfter))

	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error":       "Too Many Requests",
		"message":     fmt.Sprintf("Rate limit exceeded. Please retry after %d seconds.", retryAfter),
		"retry_after": retryAfter,
	})
}

// GetIPLimiter returns the IP-based limiter for stats access.
func (m *Middleware) GetIPLimiter() *SlidingWindowLimiter {
	return m.ipLimiter
}

// GetUserLimiter returns the user-based limiter for stats access.
func (m *Middleware) GetUserLimiter() *SlidingWindowLimiter {
	return m.userLimiter
}

// GetGlobalLimiter returns the global limiter for stats access.
func (m *Middleware) GetGlobalLimiter() *SlidingWindowLimiter {
	return m.globalLimiter
}

// isValidAPIKey validates API key format to prevent Redis key injection.
// Valid keys contain only alphanumeric characters, hyphens, underscores, and dots.
func isValidAPIKey(key string) bool {
	if len(key) == 0 || len(key) > 255 {
		return false
	}
	for _, r := range key {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}
