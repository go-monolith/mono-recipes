package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-monolith/mono"
	"github.com/redis/go-redis/v9"

	"github.com/example/rate-limiting-middleware/middleware/ratelimit"
	"github.com/example/rate-limiting-middleware/modules/api"
	apicontract "github.com/example/rate-limiting-middleware/modules/api/contract"
)

// TestRateLimitedAppIntegration runs the application as main.go wires it
// (middleware first, then the api module, strict module boundaries) and checks
// that services registered through typed contracts are wrapped by the rate
// limiting middleware. It needs Redis: set REDIS_ADDR (for example after
// `docker-compose up -d redis`), otherwise the test is skipped.
func TestRateLimitedAppIntegration(t *testing.T) {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("REDIS_ADDR not set; start Redis (docker-compose up -d redis) and set REDIS_ADDR=localhost:6379 to run")
	}
	pingRedis(t, redisAddr)

	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithNATSPort(freePort(t)),
		// Default (non-strict) mode on purpose: violations are recorded in
		// app.ModuleBoundaries() instead of rejected, so the report check below
		// is meaningful. main.go runs strict once this report is clean.
	)
	if err != nil {
		t.Fatal(err)
	}

	limiter, err := ratelimit.New(
		ratelimit.WithRedisAddr(redisAddr),
		// A key prefix of its own, so reruns and the demo do not share counters.
		ratelimit.WithKeyPrefix(fmt.Sprintf("ratelimit-test:%d:", time.Now().UnixNano())),
		ratelimit.WithServiceLimit(apicontract.CreateOrder.Name, 2, time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Register(limiter); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(api.NewModule(app.Logger())); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })

	services := app.Services("api")
	order := apicontract.OrderRequest{ProductID: "prod-123", Quantity: 1, Price: 9.99}
	for i := 1; i <= 2; i++ {
		if _, err := apicontract.CreateOrder.Call(ctx, services, order); err != nil {
			t.Fatalf("request %d within the limit failed: %v", i, err)
		}
	}
	_, err = apicontract.CreateOrder.Call(ctx, services, order)
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("request 3 over the limit of 2: err = %v, want rate limit exceeded", err)
	}

	// Other services keep their own (default) limit.
	if _, err := apicontract.GetData.Call(ctx, services, apicontract.GetDataRequest{}); err != nil {
		t.Fatalf("get-data: %v", err)
	}

	for _, r := range app.ModuleBoundaries() {
		if len(r.Violations) != 0 {
			t.Errorf("module %s: boundary violations: %+v", r.Module, r.Violations)
		}
	}
}

// pingRedis fails the test when REDIS_ADDR is set but Redis does not answer.
func pingRedis(t *testing.T, addr string) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second})
	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("REDIS_ADDR=%s is set but Redis is not reachable: %v", addr, err)
	}
}

// freePort returns a TCP port that is free right now, so the test does not
// collide with a running demo binary on the default 4222.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}
