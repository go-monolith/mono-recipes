// Package ratelimit provides domain types and interfaces for rate limiting.
package ratelimit

import (
	"context"
	"time"
)

// Config holds rate limiting configuration.
type Config struct {
	// RequestsPerWindow is the maximum number of requests allowed in the window.
	RequestsPerWindow int
	// WindowSize is the duration of the sliding window.
	WindowSize time.Duration
}

// Result represents the outcome of a rate limit check.
type Result struct {
	// Allowed indicates whether the request is allowed.
	Allowed bool
	// Remaining is the number of requests remaining in the current window.
	Remaining int
	// ResetAt is when the rate limit window resets.
	ResetAt time.Time
	// RetryAfter is the duration to wait before retrying (only set when not allowed).
	RetryAfter time.Duration
}

// Limiter is the interface for rate limiting implementations.
type Limiter interface {
	// Allow checks if a request identified by key is allowed under the rate limit.
	// It returns the result of the check and any error encountered.
	Allow(ctx context.Context, key string) (*Result, error)

	// Close releases any resources held by the limiter.
	Close() error
}

// KeyExtractor is a function that extracts a rate limit key from request context.
// It returns the key and a boolean indicating if extraction was successful.
type KeyExtractor func(ctx context.Context) (string, bool)

// MiddlewareConfig configures the rate limiting middleware.
type MiddlewareConfig struct {
	// IPConfig is the rate limit configuration for IP-based limiting.
	IPConfig Config
	// UserConfig is the rate limit configuration for authenticated user limiting.
	UserConfig Config
	// GlobalConfig is the fallback rate limit configuration.
	GlobalConfig Config
	// SkipFailedRequests skips rate limiting for failed requests (4xx, 5xx).
	SkipFailedRequests bool
	// SkipSuccessfulRequests skips rate limiting for successful requests.
	SkipSuccessfulRequests bool
	// KeyPrefix is the prefix for all rate limit keys in Redis.
	KeyPrefix string
}

// DefaultIPConfig returns the default IP-based rate limit configuration.
// 100 requests per minute as specified in the requirements.
func DefaultIPConfig() Config {
	return Config{
		RequestsPerWindow: 100,
		WindowSize:        time.Minute,
	}
}

// DefaultUserConfig returns the default user-based rate limit configuration.
// 1000 requests per minute for authenticated users.
func DefaultUserConfig() Config {
	return Config{
		RequestsPerWindow: 1000,
		WindowSize:        time.Minute,
	}
}

// DefaultGlobalConfig returns the default global rate limit configuration.
// Acts as a safety net with higher limits.
func DefaultGlobalConfig() Config {
	return Config{
		RequestsPerWindow: 10000,
		WindowSize:        time.Minute,
	}
}

// DefaultMiddlewareConfig returns the default middleware configuration.
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		IPConfig:     DefaultIPConfig(),
		UserConfig:   DefaultUserConfig(),
		GlobalConfig: DefaultGlobalConfig(),
		KeyPrefix:    "ratelimit:",
	}
}
