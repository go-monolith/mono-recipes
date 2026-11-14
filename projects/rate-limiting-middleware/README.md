# Rate Limiting Middleware Recipe

This recipe demonstrates how to implement **rate limiting** as a middleware module in the mono framework (mono **v0.1.1**). The middleware intercepts service registrations and wraps request-reply handlers with per-client, per-service rate limits using Redis. The API module registers its services through typed contracts and knows nothing about rate limiting.

## Key Concepts

### MiddlewareModule Pattern

The mono framework provides a `MiddlewareModule` interface that enables cross-cutting concerns like rate limiting, logging, and authentication. Middleware modules:

1. **Start before** regular modules
2. **Stop after** regular modules
3. **Intercept** service registrations to wrap handlers

```go
type MiddlewareModule interface {
    Module  // Name(), Start(), Stop()
    OnModuleLifecycle(ctx context.Context, event ModuleLifecycleEvent) ModuleLifecycleEvent
    OnServiceRegistration(ctx context.Context, reg ServiceRegistration) ServiceRegistration
    OnConfigurationChange(ctx context.Context, event ConfigurationEvent) ConfigurationEvent
    OnOutgoingMessage(octx OutgoingMessageContext) OutgoingMessageContext
    OnEventConsumerRegistration(ctx context.Context, entry EventConsumerEntry) EventConsumerEntry
    OnEventStreamConsumerRegistration(ctx context.Context, entry EventStreamConsumerEntry) EventStreamConsumerEntry
}
```

### Sliding Window Rate Limiting

This recipe uses a **sliding window** algorithm with Redis sorted sets for accurate rate limiting:

- Each request is added to a sorted set with its timestamp
- Expired entries (outside the window) are removed
- Request count is checked against the limit
- Atomic operations via Lua script ensure thread safety

## Project Structure

```
rate-limiting-middleware/
├── main.go                       # Application entry point (strict module boundaries)
├── integration_test.go           # Real-app test: middleware + api (needs REDIS_ADDR)
├── go.mod
├── docker-compose.yml            # Redis container
├── demo.sh                       # nats CLI demo
├── middleware/
│   └── ratelimit/
│       ├── config.go             # Configuration and options
│       ├── limiter.go            # Redis sliding window limiter
│       ├── middleware.go         # MiddlewareModule implementation
│       └── *_test.go             # Unit tests (no Redis needed)
└── modules/
    └── api/
        ├── contract/
        │   └── contract.go       # Published port: DTOs + typed service contracts
        ├── module.go             # Domain handlers + RegisterServices via contracts
        ├── handlers_test.go      # Handler unit tests
        └── module_test.go        # monotest.CheckModule conformance test
```

## Features

- **Per-client, per-service rate limiting**: Different limits for different services
- **Redis-based**: Distributed rate limiting across multiple instances
- **Sliding window algorithm**: More accurate than fixed windows
- **Fail-open**: Allows requests on Redis errors (configurable)
- **Configurable client ID extraction**: From headers or fallback to default

## Configuration

The middleware supports the following options:

```go
ratelimit.New(
    ratelimit.WithRedisAddr("localhost:6379"),
    ratelimit.WithRedisPassword(""),
    ratelimit.WithRedisDB(0),

    // Default limit for all services
    ratelimit.WithDefaultLimit(100, time.Minute),

    // Per-service limits, keyed by service name (taken from the contracts)
    ratelimit.WithServiceLimit(apicontract.CreateOrder.Name, 50, time.Minute),
    ratelimit.WithServiceLimit(apicontract.GetStatus.Name, 200, time.Minute),

    // Key prefix for Redis keys
    ratelimit.WithKeyPrefix("ratelimit:"),

    // Client ID extraction from headers (fallback client ID: "anonymous")
    ratelimit.WithClientIDHeader("X-Client-ID"),
)
```

## Running the Demo

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- NATS CLI (`nats`)

### Start Redis

```bash
docker-compose up -d
```

### Run the Application

```bash
go build -o bin/rate-limiting-middleware . && ./bin/rate-limiting-middleware
# or: go run .
```

`REDIS_ADDR` (default `localhost:6379`) and `REDIS_PASSWORD` configure Redis. The middleware's `Start` pings Redis, so the application does not start without it.

### Test Rate Limiting

```bash
# Test with client ID (each client has separate limits)
nats request services.api.get-data '{}' --header X-Client-ID:client1

# Make multiple requests to hit the limit
for i in {1..110}; do
  nats request services.api.get-data '{}' --header X-Client-ID:client2
done

# Test service with lower limit (50 req/min)
nats request services.api.create-order '{"product_id":"prod-123","quantity":1,"price":29.99}' --header X-Client-ID:client1

# Test service with higher limit (200 req/min)
nats request services.api.get-status '{}' --header X-Client-ID:client1
```

`get-data` and `get-status` take no parameters and also accept an empty body (`nats request services.api.get-data ""`).

### Using the Demo Script

```bash
./demo.sh
```

## How It Works

### 1. Middleware Registration Order

Middleware must be registered **before** regular modules:

```go
// main.go
app.Register(rateLimitMiddleware)  // First: middleware
app.Register(apiModule)            // Then: regular modules
```

### 2. Service Registration Interception

The API module registers its services through typed contracts declared in `modules/api/contract`:

```go
// modules/api/contract/contract.go
var CreateOrder = helper.RequestReplyService[OrderRequest, OrderResponse]("api", "create-order")

// modules/api/module.go
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    if err := contract.GetData.Register(container, m.handleGetData); err != nil {
        return err
    }
    if err := contract.CreateOrder.Register(container, m.handleCreateOrder); err != nil {
        return err
    }
    return contract.GetStatus.Register(container, m.handleGetStatus)
}
```

`Register` wraps the typed handler into a plain `RequestReplyHandler` (JSON decode, call, JSON encode) and registers it on the container, so every registration still passes through the middleware chain. The middleware intercepts each one:

```go
func (m *Middleware) OnServiceRegistration(
    _ context.Context,
    reg types.ServiceRegistration,
) types.ServiceRegistration {
    // Only wrap request-reply services
    if reg.Type != types.ServiceTypeRequestReply || reg.RequestHandler == nil {
        return reg
    }

    original := reg.RequestHandler
    serviceName := reg.Name

    // Wrap with rate limiting
    reg.RequestHandler = func(ctx context.Context, req *types.Msg) ([]byte, error) {
        clientID := m.extractClientID(req)
        key := fmt.Sprintf("%s:%s", serviceName, clientID)

        result, err := m.limiter.Allow(ctx, key, limit, window)
        if err != nil {
            // Fail-open: allow on Redis error
            return original(ctx, req)
        }

        if !result.Allowed {
            return nil, &RateLimitError{...} // becomes an error reply
        }

        return original(ctx, req)
    }

    return reg // every other field, including reg.Options, is unchanged
}
```

The middleware only replaces `reg.RequestHandler` on the registration it receives and returns it, so per-service worker pool options (`reg.Options`, mono v0.0.15+) are preserved. Since mono v0.0.15, request-reply handlers run concurrently on a worker pool; the Lua script keeps the check-and-increment atomic, and the handler context ends at the caller's deadline, which also bounds the Redis call.

### 3. Sliding Window Algorithm

The Lua script ensures atomic operations:

```lua
-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current requests
local current = redis.call('ZCARD', key)

if current < limit then
    -- Add new request with timestamp
    redis.call('ZADD', key, now, now .. ':' .. math.random())
    redis.call('EXPIRE', key, window_seconds)
    return {1, limit - current - 1, 0}  -- allowed, remaining, reset_at
else
    -- Rate limited
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local reset_at = oldest[2] + window_seconds * 1000
    return {0, 0, reset_at}  -- denied, remaining, reset_at
end
```

## Rate Limit Response

When the rate limit is exceeded, the handler returns a `*ratelimit.RateLimitError` and the framework sends an error reply with an empty body and these headers:

```
Mono-Error: true
Mono-Error-Message: rate limit exceeded for service create-order
Mono-Error-Type: ratelimit
```

A Go caller gets it as the error of the contract call (`contract api/create-order (request_reply): ... rate limit exceeded for service create-order`). The limit, remaining count and reset time are logged by the middleware.

## Module Boundaries

The recipe follows the module boundaries path of mono v0.1.x:

- **Contract package**: `modules/api/contract` holds the DTOs and the typed contracts (`GetData`, `CreateOrder`, `GetStatus`). It imports only the standard library and `mono/pkg/helper`, and is the only api package other code may import. Subjects stay `services.api.<service>`.
- **No consumers inside the app**: the services are called from outside (nats CLI, `demo.sh`), so the api module has no `ConsumeServices()` and there are no ports or adapters. The provider-owned `ServiceAdapter` of earlier versions was removed; a module that needs these services would own its port and adapter and call `contract.X.Call`.
- **Strict mode**: `main.go` runs with `mono.WithStrictModuleBoundaries()`.
- **Tests**:

```bash
go test ./...                                            # unit + monotest.CheckModule (no Redis needed)
REDIS_ADDR=localhost:6379 go test -run Integration .     # real app: middleware wraps the contract services
```

`monotest` has no middleware chain, so the conformance test (`modules/api/module_test.go`) checks the api module alone; the middleware is covered by its unit tests and by `integration_test.go`, which is skipped when `REDIS_ADDR` is not set.

## Best Practices

1. **Register middleware first**: Middleware must be registered before the modules it needs to intercept
2. **Fail-open vs fail-closed**: This implementation fails open on Redis errors; adjust based on your security requirements
3. **Client identification**: Use authenticated user IDs or API keys rather than relying on headers in production
4. **Distributed rate limiting**: Redis ensures rate limits work across multiple application instances
5. **Monitor rate limit events**: Log rate limit hits for capacity planning

## Related Middleware

The mono framework ships ready-made middleware in `github.com/go-monolith/mono/middleware`:

- `middleware/requestid` - Add request tracing
- `middleware/accesslog` - Request/response logging
- `middleware/audit` - Tamper-evident audit logging
