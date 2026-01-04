package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter implements sliding window rate limiting using Redis.
type Limiter struct {
	client    *redis.Client
	keyPrefix string
}

// NewLimiter creates a new rate limiter with Redis backend.
func NewLimiter(client *redis.Client, keyPrefix string) *Limiter {
	return &Limiter{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// RateLimitResult contains the result of a rate limit check.
type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
	Limit     int
}

// Allow checks if a request is allowed under the rate limit.
// Uses sliding window algorithm with Redis sorted sets.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window)

	redisKey := l.keyPrefix + key

	// Use a Lua script for atomic sliding window rate limiting
	// This ensures thread-safety and consistency
	// Uses INCR counter to generate unique member values (avoids math.random() issues)
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_ms = tonumber(ARGV[4])

		-- Remove expired entries
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests in window
		local current = redis.call('ZCARD', key)

		if current < limit then
			-- Generate unique member using atomic counter
			local counter = redis.call('INCR', key .. ':counter')
			redis.call('ZADD', key, now, now .. ':' .. counter)
			-- Set expiry on the key (convert ms to seconds, round up)
			local expire_seconds = math.ceil(window_ms / 1000)
			redis.call('EXPIRE', key, expire_seconds)
			redis.call('EXPIRE', key .. ':counter', expire_seconds)
			return {1, limit - current - 1, 0}
		else
			-- Get the oldest entry to calculate reset time
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local reset_at = 0
			if oldest and #oldest >= 2 then
				reset_at = tonumber(oldest[2]) + window_ms
			end
			return {0, 0, reset_at}
		end
	`)

	// Convert times to milliseconds for precision (consistent time units)
	nowMs := now.UnixMilli()
	windowStartMs := windowStart.UnixMilli()
	windowMs := window.Milliseconds()

	result, err := script.Run(ctx, l.client, []string{redisKey}, nowMs, windowStartMs, limit, windowMs).Int64Slice()
	if err != nil {
		return nil, fmt.Errorf("redis script error: %w", err)
	}

	// Validate response length to prevent panic
	if len(result) != 3 {
		return nil, fmt.Errorf("unexpected Redis response length: %d", len(result))
	}

	allowed := result[0] == 1
	remaining := int(result[1])
	resetAtMs := result[2]

	var resetAt time.Time
	if resetAtMs > 0 {
		resetAt = time.UnixMilli(resetAtMs)
	} else {
		resetAt = now.Add(window)
	}

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
		Limit:     limit,
	}, nil
}

// Reset clears the rate limit for a specific key.
func (l *Limiter) Reset(ctx context.Context, key string) error {
	redisKey := l.keyPrefix + key
	return l.client.Del(ctx, redisKey).Err()
}

// GetStats returns current rate limit stats for a key.
func (l *Limiter) GetStats(ctx context.Context, key string, window time.Duration) (int, error) {
	redisKey := l.keyPrefix + key
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove expired entries first
	_, err := l.client.ZRemRangeByScore(ctx, redisKey, "-inf", fmt.Sprintf("%d", windowStart.UnixMilli())).Result()
	if err != nil {
		return 0, err
	}

	// Count current requests
	count, err := l.client.ZCard(ctx, redisKey).Result()
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
