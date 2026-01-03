package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/example/rate-limiting-demo/domain/ratelimit"
	"github.com/redis/go-redis/v9"
)

// TestSlidingWindowLimiter_Allow tests the basic rate limiting behavior.
func TestSlidingWindowLimiter_Allow(t *testing.T) {
	// Skip if Redis is not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	// Clean up test keys before and after
	testPrefix := "test:ratelimit:"
	defer client.Del(ctx, testPrefix+"test-key")

	config := ratelimit.Config{
		RequestsPerWindow: 5,
		WindowSize:        time.Minute,
	}

	limiter := NewSlidingWindowLimiter(client, config, testPrefix)

	// Test that first 5 requests are allowed
	for i := 0; i < 5; i++ {
		result, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if result.Remaining != 5-i-1 {
			t.Errorf("Expected %d remaining, got %d", 5-i-1, result.Remaining)
		}
	}

	// 6th request should be denied
	result, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("6th request should be denied")
	}
	if result.Remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", result.Remaining)
	}
	if result.RetryAfter <= 0 {
		t.Error("RetryAfter should be positive")
	}
}

// TestSlidingWindowLimiter_DifferentKeys tests that different keys have separate limits.
func TestSlidingWindowLimiter_DifferentKeys(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	testPrefix := "test:ratelimit:diffkeys:"
	defer client.Del(ctx, testPrefix+"key1", testPrefix+"key2")

	config := ratelimit.Config{
		RequestsPerWindow: 3,
		WindowSize:        time.Minute,
	}

	limiter := NewSlidingWindowLimiter(client, config, testPrefix)

	// Exhaust limit for key1
	for i := 0; i < 3; i++ {
		result, _ := limiter.Allow(ctx, "key1")
		if !result.Allowed {
			t.Errorf("key1 request %d should be allowed", i+1)
		}
	}

	// key1 should now be rate limited
	result, _ := limiter.Allow(ctx, "key1")
	if result.Allowed {
		t.Error("key1 should be rate limited")
	}

	// key2 should still be allowed (independent limit)
	result, _ = limiter.Allow(ctx, "key2")
	if !result.Allowed {
		t.Error("key2 should be allowed (independent limit)")
	}
}

// TestSlidingWindowLimiter_GetStats tests the statistics retrieval.
func TestSlidingWindowLimiter_GetStats(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	testPrefix := "test:ratelimit:stats:"
	defer client.Del(ctx, testPrefix+"stats-key")

	config := ratelimit.Config{
		RequestsPerWindow: 10,
		WindowSize:        time.Minute,
	}

	limiter := NewSlidingWindowLimiter(client, config, testPrefix)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, "stats-key")
	}

	// Get stats
	stats, err := limiter.GetStats(ctx, "stats-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	currentCount := stats["current_count"].(int64)
	if currentCount != 3 {
		t.Errorf("Expected current_count=3, got %d", currentCount)
	}

	remaining := stats["remaining"].(int)
	if remaining != 7 {
		t.Errorf("Expected remaining=7, got %d", remaining)
	}

	limit := stats["limit"].(int)
	if limit != 10 {
		t.Errorf("Expected limit=10, got %d", limit)
	}
}

// TestSlidingWindowLimiter_WindowExpiry tests that the window expires correctly.
func TestSlidingWindowLimiter_WindowExpiry(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}

	testPrefix := "test:ratelimit:expiry:"
	defer client.Del(ctx, testPrefix+"expiry-key")

	// Use a very short window for testing
	config := ratelimit.Config{
		RequestsPerWindow: 2,
		WindowSize:        100 * time.Millisecond,
	}

	limiter := NewSlidingWindowLimiter(client, config, testPrefix)

	// Exhaust the limit
	limiter.Allow(ctx, "expiry-key")
	limiter.Allow(ctx, "expiry-key")

	// Should be rate limited
	result, _ := limiter.Allow(ctx, "expiry-key")
	if result.Allowed {
		t.Error("Should be rate limited")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	result, _ = limiter.Allow(ctx, "expiry-key")
	if !result.Allowed {
		t.Error("Should be allowed after window expiry")
	}
}

// TestSlidingWindowLimiter_GetConfig tests config retrieval.
func TestSlidingWindowLimiter_GetConfig(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	config := ratelimit.Config{
		RequestsPerWindow: 100,
		WindowSize:        5 * time.Minute,
	}

	limiter := NewSlidingWindowLimiter(client, config, "test:")

	retrievedConfig := limiter.GetConfig()
	if retrievedConfig.RequestsPerWindow != 100 {
		t.Errorf("Expected RequestsPerWindow=100, got %d", retrievedConfig.RequestsPerWindow)
	}
	if retrievedConfig.WindowSize != 5*time.Minute {
		t.Errorf("Expected WindowSize=5m, got %v", retrievedConfig.WindowSize)
	}
}

// TestSlidingWindowLimiter_Close tests the close method.
func TestSlidingWindowLimiter_Close(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	config := ratelimit.DefaultIPConfig()
	limiter := NewSlidingWindowLimiter(client, config, "test:")

	// Close should not return an error
	err := limiter.Close()
	if err != nil {
		t.Errorf("Unexpected error on close: %v", err)
	}
}
