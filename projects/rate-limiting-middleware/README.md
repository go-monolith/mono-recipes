# Rate Limiting Middleware Recipe

This recipe demonstrates how to implement **rate limiting** as a middleware module in the mono framework. The middleware intercepts service registrations and wraps request-reply handlers with per-client, per-service rate limits using Redis.

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
├── main.go                       # Application entry point
├── go.mod
├── docker-compose.yml            # Redis container
├── middleware/
│   └── ratelimit/
│       ├── config.go             # Configuration and options
│       ├── limiter.go            # Redis sliding window limiter
│       └── middleware.go         # MiddlewareModule implementation
└── modules/
    └── api/
        ├── module.go             # API services (request-reply)
        └── types.go              # Request/response types
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

    // Per-service limits
    ratelimit.WithServiceLimit("create-order", 50, time.Minute),
    ratelimit.WithServiceLimit("get-status", 200, time.Minute),

    // Key prefix for Redis keys
    ratelimit.WithKeyPrefix("ratelimit:"),

    // Client ID extraction from headers
    ratelimit.WithClientIDHeader("X-Client-ID"),
    ratelimit.WithFallbackClientID("anonymous"),
)
```

## Running the Demo

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- NATS CLI (`nats`)

### Start Redis

```bash
docker-compose up -d
```

### Run the Application

```bash
go run main.go
```

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
app.Register(apiModule)             // Then: regular modules
```

### 2. Service Registration Interception

When the API module registers its services, the middleware intercepts each registration:

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
            return rateLimitErrorResponse, nil
        }

        return original(ctx, req)
    }

    return reg
}
```

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

When rate limit is exceeded:

```json
{
    "error": "rate limit exceeded for service get-data",
    "remaining": 0,
    "reset_at": "2024-01-15T10:30:00Z",
    "limit": 100
}
```

## Best Practices

1. **Register middleware first**: Middleware must be registered before the modules it needs to intercept
2. **Fail-open vs fail-closed**: This implementation fails open on Redis errors; adjust based on your security requirements
3. **Client identification**: Use authenticated user IDs or API keys rather than relying on headers in production
4. **Distributed rate limiting**: Redis ensures rate limits work across multiple application instances
5. **Monitor rate limit events**: Log rate limit hits for capacity planning

## Related Recipes

- [Request ID Middleware](../request-id-middleware/) - Add request tracing
- [Access Log Middleware](../access-log-middleware/) - Request/response logging
- [Audit Trail Middleware](../audit-middleware/) - Tamper-evident audit logging
