// Package ratelimit provides a Redis-based sliding window rate limiter implementation.
package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/example/rate-limiting-demo/domain/ratelimit"
	"github.com/redis/go-redis/v9"
)

// SlidingWindowLimiter implements a sliding window rate limiter using Redis.
// It uses a sorted set to track request timestamps and calculates the count
// of requests within the sliding window.
type SlidingWindowLimiter struct {
	client *redis.Client
	config ratelimit.Config
	prefix string
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter.
func NewSlidingWindowLimiter(client *redis.Client, config ratelimit.Config, prefix string) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		client: client,
		config: config,
		prefix: prefix,
	}
}

// Allow checks if a request is allowed under the rate limit using a sliding window algorithm.
// The algorithm works as follows:
// 1. Remove all entries outside the current window
// 2. Count remaining entries
// 3. If count < limit, add new entry and allow
// 4. Otherwise, deny and calculate retry-after
func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (*ratelimit.Result, error) {
	now := time.Now()
	windowStart := now.Add(-l.config.WindowSize)
	redisKey := l.prefix + key

	// Use a Lua script for atomic operations
	script := redis.NewScript(`
		local key = KEYS[1]
		local counter_key = KEYS[2]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_size_ms = tonumber(ARGV[4])

		-- Remove old entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current entries
		local count = redis.call('ZCARD', key)

		if count < limit then
			-- Use atomic counter for unique member ID (prevents collision)
			local counter = redis.call('INCR', counter_key)
			-- Add new entry with current timestamp as score
			redis.call('ZADD', key, now, now .. ':' .. counter)
			-- Set expiry on both keys to clean up automatically
			redis.call('PEXPIRE', key, window_size_ms)
			redis.call('PEXPIRE', counter_key, window_size_ms)
			return {1, limit - count - 1, 0}
		else
			-- Get the oldest entry to calculate retry-after
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local retry_after = 0
			if #oldest >= 2 then
				retry_after = oldest[2] + window_size_ms - now
			end
			return {0, 0, retry_after}
		end
	`)

	nowMs := now.UnixMilli()
	windowStartMs := windowStart.UnixMilli()
	windowSizeMs := l.config.WindowSize.Milliseconds()
	counterKey := redisKey + ":counter"

	result, err := script.Run(ctx, l.client, []string{redisKey, counterKey},
		nowMs,
		windowStartMs,
		l.config.RequestsPerWindow,
		windowSizeMs,
	).Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to run rate limit script: %w", err)
	}

	// Safely extract results with type checks
	if len(result) < 3 {
		return nil, fmt.Errorf("unexpected result length: %d", len(result))
	}

	allowedVal, ok := result[0].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected type for allowed: %T", result[0])
	}
	remainingVal, ok := result[1].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected type for remaining: %T", result[1])
	}
	retryAfterMs, ok := result[2].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected type for retry_after: %T", result[2])
	}

	allowed := allowedVal == 1
	remaining := int(remainingVal)

	res := &ratelimit.Result{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   now.Add(l.config.WindowSize),
	}

	if !allowed && retryAfterMs > 0 {
		res.RetryAfter = time.Duration(retryAfterMs) * time.Millisecond
	}

	return res, nil
}

// Close releases any resources (Redis client is managed externally).
func (l *SlidingWindowLimiter) Close() error {
	return nil
}

// GetConfig returns the limiter's configuration.
func (l *SlidingWindowLimiter) GetConfig() ratelimit.Config {
	return l.config
}

// GetStats returns statistics about the rate limiter for a specific key.
func (l *SlidingWindowLimiter) GetStats(ctx context.Context, key string) (map[string]interface{}, error) {
	redisKey := l.prefix + key
	now := time.Now()
	windowStart := now.Add(-l.config.WindowSize)

	// Get count of requests in current window
	count, err := l.client.ZCount(ctx, redisKey, strconv.FormatInt(windowStart.UnixMilli(), 10), "+inf").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get request count: %w", err)
	}

	return map[string]interface{}{
		"key":                redisKey,
		"current_count":      count,
		"limit":              l.config.RequestsPerWindow,
		"remaining":          l.config.RequestsPerWindow - int(count),
		"window_size_seconds": l.config.WindowSize.Seconds(),
	}, nil
}
