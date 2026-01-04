package ratelimit

import (
	"time"
)

// Config holds rate limiter configuration.
type Config struct {
	// RedisAddr is the Redis server address (e.g., "localhost:6379")
	RedisAddr string

	// RedisPassword is the Redis authentication password (optional)
	RedisPassword string

	// RedisDB is the Redis database number (default: 0)
	RedisDB int

	// DefaultLimit is the default rate limit if no service-specific limit is configured
	DefaultLimit int

	// DefaultWindow is the default time window for rate limiting
	DefaultWindow time.Duration

	// ServiceLimits maps service names to their specific rate limits
	ServiceLimits map[string]ServiceLimit

	// KeyPrefix is the prefix for Redis keys (default: "ratelimit:")
	KeyPrefix string

	// ClientIDHeader is the header name to extract client ID from (default: "X-Client-ID")
	ClientIDHeader string

	// FallbackClientID is used when no client ID is found in request
	FallbackClientID string
}

// ServiceLimit defines rate limits for a specific service.
type ServiceLimit struct {
	// Limit is the maximum number of requests allowed in the window
	Limit int

	// Window is the time window for the rate limit
	Window time.Duration
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		RedisAddr:        "localhost:6379",
		RedisPassword:    "",
		RedisDB:          0,
		DefaultLimit:     100,
		DefaultWindow:    time.Minute,
		ServiceLimits:    make(map[string]ServiceLimit),
		KeyPrefix:        "ratelimit:",
		ClientIDHeader:   "X-Client-ID",
		FallbackClientID: "anonymous",
	}
}

// Option is a function that modifies Config.
type Option func(*Config)

// WithRedisAddr sets the Redis server address.
func WithRedisAddr(addr string) Option {
	return func(c *Config) {
		c.RedisAddr = addr
	}
}

// WithRedisPassword sets the Redis authentication password.
func WithRedisPassword(password string) Option {
	return func(c *Config) {
		c.RedisPassword = password
	}
}

// WithRedisDB sets the Redis database number.
func WithRedisDB(db int) Option {
	return func(c *Config) {
		c.RedisDB = db
	}
}

// WithDefaultLimit sets the default rate limit.
func WithDefaultLimit(limit int, window time.Duration) Option {
	return func(c *Config) {
		c.DefaultLimit = limit
		c.DefaultWindow = window
	}
}

// WithServiceLimit sets a specific rate limit for a service.
func WithServiceLimit(serviceName string, limit int, window time.Duration) Option {
	return func(c *Config) {
		c.ServiceLimits[serviceName] = ServiceLimit{
			Limit:  limit,
			Window: window,
		}
	}
}

// WithKeyPrefix sets the Redis key prefix.
func WithKeyPrefix(prefix string) Option {
	return func(c *Config) {
		c.KeyPrefix = prefix
	}
}

// WithClientIDHeader sets the header name for client ID extraction.
func WithClientIDHeader(header string) Option {
	return func(c *Config) {
		c.ClientIDHeader = header
	}
}
