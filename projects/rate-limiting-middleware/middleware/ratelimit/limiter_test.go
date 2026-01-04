package ratelimit

import (
	"testing"
	"time"
)

func TestRateLimitResult(t *testing.T) {
	result := &RateLimitResult{
		Allowed:   true,
		Remaining: 99,
		ResetAt:   time.Now().Add(time.Minute),
		Limit:     100,
	}

	if !result.Allowed {
		t.Error("expected Allowed to be true")
	}
	if result.Remaining != 99 {
		t.Errorf("expected Remaining 99, got %d", result.Remaining)
	}
	if result.Limit != 100 {
		t.Errorf("expected Limit 100, got %d", result.Limit)
	}
}

func TestNewLimiter(t *testing.T) {
	// NewLimiter should work with nil client for unit testing
	limiter := NewLimiter(nil, "test:")

	if limiter == nil {
		t.Fatal("NewLimiter returned nil")
	}
	if limiter.keyPrefix != "test:" {
		t.Errorf("expected keyPrefix 'test:', got %q", limiter.keyPrefix)
	}
}

func TestNewLimiter_EmptyPrefix(t *testing.T) {
	limiter := NewLimiter(nil, "")

	if limiter == nil {
		t.Fatal("NewLimiter returned nil")
	}
	if limiter.keyPrefix != "" {
		t.Errorf("expected empty keyPrefix, got %q", limiter.keyPrefix)
	}
}

// Note: Integration tests for Allow(), Reset(), and GetStats()
// require a running Redis instance. These would be implemented
// as integration tests with testcontainers or a test Redis instance.
//
// Example integration test structure:
//
// func TestLimiter_Allow_Integration(t *testing.T) {
//     if testing.Short() {
//         t.Skip("skipping integration test in short mode")
//     }
//
//     ctx := context.Background()
//     client := redis.NewClient(&redis.Options{
//         Addr: "localhost:6379",
//     })
//     defer client.Close()
//
//     limiter := NewLimiter(client, "test:")
//
//     // Test allowing requests within limit
//     for i := 0; i < 10; i++ {
//         result, err := limiter.Allow(ctx, "test-key", 100, time.Minute)
//         if err != nil {
//             t.Fatalf("Allow() error: %v", err)
//         }
//         if !result.Allowed {
//             t.Errorf("request %d should be allowed", i)
//         }
//     }
// }
