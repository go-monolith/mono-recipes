package ratelimit

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("expected RedisAddr 'localhost:6379', got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "" {
		t.Errorf("expected empty RedisPassword, got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 0 {
		t.Errorf("expected RedisDB 0, got %d", cfg.RedisDB)
	}
	if cfg.DefaultLimit != 100 {
		t.Errorf("expected DefaultLimit 100, got %d", cfg.DefaultLimit)
	}
	if cfg.DefaultWindow != time.Minute {
		t.Errorf("expected DefaultWindow 1m, got %v", cfg.DefaultWindow)
	}
	if cfg.KeyPrefix != "ratelimit:" {
		t.Errorf("expected KeyPrefix 'ratelimit:', got %q", cfg.KeyPrefix)
	}
	if cfg.ClientIDHeader != "X-Client-ID" {
		t.Errorf("expected ClientIDHeader 'X-Client-ID', got %q", cfg.ClientIDHeader)
	}
	if cfg.FallbackClientID != "anonymous" {
		t.Errorf("expected FallbackClientID 'anonymous', got %q", cfg.FallbackClientID)
	}
	if cfg.ServiceLimits == nil {
		t.Error("expected ServiceLimits to be initialized")
	}
}

func TestWithRedisAddr(t *testing.T) {
	cfg := DefaultConfig()
	WithRedisAddr("redis.example.com:6380")(&cfg)

	if cfg.RedisAddr != "redis.example.com:6380" {
		t.Errorf("expected RedisAddr 'redis.example.com:6380', got %q", cfg.RedisAddr)
	}
}

func TestWithRedisPassword(t *testing.T) {
	cfg := DefaultConfig()
	WithRedisPassword("secret123")(&cfg)

	if cfg.RedisPassword != "secret123" {
		t.Errorf("expected RedisPassword 'secret123', got %q", cfg.RedisPassword)
	}
}

func TestWithRedisDB(t *testing.T) {
	cfg := DefaultConfig()
	WithRedisDB(5)(&cfg)

	if cfg.RedisDB != 5 {
		t.Errorf("expected RedisDB 5, got %d", cfg.RedisDB)
	}
}

func TestWithDefaultLimit(t *testing.T) {
	cfg := DefaultConfig()
	WithDefaultLimit(200, 30*time.Second)(&cfg)

	if cfg.DefaultLimit != 200 {
		t.Errorf("expected DefaultLimit 200, got %d", cfg.DefaultLimit)
	}
	if cfg.DefaultWindow != 30*time.Second {
		t.Errorf("expected DefaultWindow 30s, got %v", cfg.DefaultWindow)
	}
}

func TestWithServiceLimit(t *testing.T) {
	cfg := DefaultConfig()
	WithServiceLimit("get-data", 50, 2*time.Minute)(&cfg)
	WithServiceLimit("create-order", 10, 10*time.Second)(&cfg)

	limit1, ok := cfg.ServiceLimits["get-data"]
	if !ok {
		t.Fatal("expected 'get-data' to be in ServiceLimits")
	}
	if limit1.Limit != 50 {
		t.Errorf("expected limit 50, got %d", limit1.Limit)
	}
	if limit1.Window != 2*time.Minute {
		t.Errorf("expected window 2m, got %v", limit1.Window)
	}

	limit2, ok := cfg.ServiceLimits["create-order"]
	if !ok {
		t.Fatal("expected 'create-order' to be in ServiceLimits")
	}
	if limit2.Limit != 10 {
		t.Errorf("expected limit 10, got %d", limit2.Limit)
	}
	if limit2.Window != 10*time.Second {
		t.Errorf("expected window 10s, got %v", limit2.Window)
	}
}

func TestWithKeyPrefix(t *testing.T) {
	cfg := DefaultConfig()
	WithKeyPrefix("myapp:limits:")(&cfg)

	if cfg.KeyPrefix != "myapp:limits:" {
		t.Errorf("expected KeyPrefix 'myapp:limits:', got %q", cfg.KeyPrefix)
	}
}

func TestWithClientIDHeader(t *testing.T) {
	cfg := DefaultConfig()
	WithClientIDHeader("X-API-Key")(&cfg)

	if cfg.ClientIDHeader != "X-API-Key" {
		t.Errorf("expected ClientIDHeader 'X-API-Key', got %q", cfg.ClientIDHeader)
	}
}

func TestMultipleOptions(t *testing.T) {
	cfg := DefaultConfig()
	opts := []Option{
		WithRedisAddr("redis:6379"),
		WithRedisPassword("pass"),
		WithRedisDB(3),
		WithDefaultLimit(500, 5*time.Minute),
		WithServiceLimit("svc1", 100, time.Minute),
		WithKeyPrefix("test:"),
		WithClientIDHeader("X-User"),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.RedisAddr != "redis:6379" {
		t.Errorf("expected RedisAddr 'redis:6379', got %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "pass" {
		t.Errorf("expected RedisPassword 'pass', got %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 3 {
		t.Errorf("expected RedisDB 3, got %d", cfg.RedisDB)
	}
	if cfg.DefaultLimit != 500 {
		t.Errorf("expected DefaultLimit 500, got %d", cfg.DefaultLimit)
	}
	if cfg.DefaultWindow != 5*time.Minute {
		t.Errorf("expected DefaultWindow 5m, got %v", cfg.DefaultWindow)
	}
	if cfg.KeyPrefix != "test:" {
		t.Errorf("expected KeyPrefix 'test:', got %q", cfg.KeyPrefix)
	}
	if cfg.ClientIDHeader != "X-User" {
		t.Errorf("expected ClientIDHeader 'X-User', got %q", cfg.ClientIDHeader)
	}
	if len(cfg.ServiceLimits) != 1 {
		t.Errorf("expected 1 service limit, got %d", len(cfg.ServiceLimits))
	}
}
