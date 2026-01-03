# Rate Limiting Demo

A demonstration of distributed rate limiting using the [go-monolith/mono](https://github.com/go-monolith/mono) framework with [Fiber](https://gofiber.io/) and [Redis](https://redis.io/).

## Why Rate Limiting?

Rate limiting is essential for protecting your APIs from abuse and ensuring fair resource allocation among users. Without rate limiting:

- **Denial of Service**: A single client can overwhelm your server with requests
- **Resource Starvation**: Heavy users consume resources that should be shared
- **Cost Overruns**: Uncontrolled API usage can lead to unexpected infrastructure costs
- **Security Vulnerabilities**: Brute force attacks become trivially easy

### Why Distributed Rate Limiting?

In-memory rate limiting works for single-instance deployments but fails when you scale horizontally:

| Aspect | In-Memory | Distributed (Redis) |
|--------|-----------|---------------------|
| **Horizontal Scaling** | ❌ Each instance has separate counters | ✅ Shared counters across instances |
| **Consistency** | ❌ Requests to different instances bypass limits | ✅ Consistent enforcement |
| **Persistence** | ❌ Lost on restart | ✅ Survives restarts |
| **Latency** | ✅ ~1μs | ⚠️ ~1ms (network round-trip) |
| **Complexity** | ✅ Simple | ⚠️ Requires Redis infrastructure |

**Use distributed rate limiting when**:
- Running multiple instances behind a load balancer
- Rate limits must survive application restarts
- Consistency across your fleet is critical

## Rate Limiting Algorithms

### Sliding Window (Used in This Demo)

The sliding window algorithm provides smooth rate limiting without the "burst at window edges" problem of fixed windows:

```
Fixed Window Problem:
|-------- Window 1 --------|-------- Window 2 --------|
                    [99 req][1 req]
                    ← User sends 100 requests at the boundary →

Sliding Window Solution:
Time: ----[current window: 60 seconds]---->
      ↑                                   ↑
      Oldest request                      Now
      (will expire)                       (new request)

Requests are tracked individually with timestamps.
The window "slides" forward continuously.
```

**How it works:**
1. Store each request timestamp in a Redis sorted set
2. Remove timestamps older than the window size
3. Count remaining entries
4. If count < limit, allow and add new timestamp
5. If count >= limit, deny and return retry-after

**Trade-offs:**

| Algorithm | Pros | Cons |
|-----------|------|------|
| **Fixed Window** | Simple, low memory | Burst at edges |
| **Sliding Window** | Smooth, accurate | Higher memory per key |
| **Token Bucket** | Allows controlled bursts | Complex to distribute |
| **Leaky Bucket** | Smoothest output rate | May drop valid requests |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Fiber HTTP Server                        │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                Rate Limiting Middleware                  ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ││
│  │  │ IP Limiter   │  │ User Limiter │  │Global Limiter│  ││
│  │  │ 100 req/min  │  │ 1000 req/min │  │10000 req/min │  ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘  ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Redis (Sliding Window)                   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  ratelimit:ip:192.168.1.1     → ZSET{timestamps}        ││
│  │  ratelimit:user:apikey:abc123 → ZSET{timestamps}        ││
│  │  ratelimit:global:all         → ZSET{timestamps}        ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Project Structure

```
rate-limiting-demo/
├── main.go                          # Application entry point
├── docker-compose.yml               # Redis container setup
├── demo.sh                          # Demo script
├── domain/
│   └── ratelimit/
│       └── types.go                 # Domain types and interfaces
└── modules/
    ├── ratelimit/
    │   ├── module.go                # Mono module for rate limiting
    │   ├── sliding_window.go        # Sliding window algorithm
    │   └── middleware.go            # Fiber middleware
    └── api/
        ├── module.go                # HTTP API module
        └── handlers.go              # HTTP handlers
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose (for Redis)

### Running the Demo

1. **Start Redis:**

```bash
docker-compose up -d
```

2. **Run the application:**

```bash
go run .
```

3. **Test the endpoints:**

```bash
# Run the demo script
./demo.sh

# Or test manually:
# Public endpoint (100 req/min by IP)
curl http://localhost:8080/api/v1/public

# Premium endpoint (1000 req/min by API key)
curl -H "X-API-Key: my-secret-key" http://localhost:8080/api/v1/premium

# Check rate limit stats
curl http://localhost:8080/api/v1/stats
```

## API Endpoints

| Endpoint | Method | Description | Rate Limit |
|----------|--------|-------------|------------|
| `/health` | GET | Health check | None |
| `/api/v1/public` | GET | Public endpoint | 100 req/min (by IP) |
| `/api/v1/premium` | GET | Premium endpoint | 1000 req/min (by API key) |
| `/api/v1/stats` | GET | Rate limit statistics | None |

### Response Headers

All rate-limited endpoints return these headers:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed per window |
| `X-RateLimit-Remaining` | Requests remaining in current window |
| `X-RateLimit-Reset` | Unix timestamp when the window resets |
| `Retry-After` | Seconds to wait (only on 429 responses) |

### Rate Limit Exceeded Response (HTTP 429)

```json
{
  "error": "Too Many Requests",
  "message": "Rate limit exceeded. Please retry after 42 seconds.",
  "retry_after": 42
}
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis server address |
| `HTTP_PORT` | `8080` | HTTP server port |

## Production Considerations

### High Availability

For production deployments, consider:

1. **Redis Sentinel or Cluster**: For Redis high availability
2. **Connection Pooling**: The go-redis client handles this automatically
3. **Circuit Breaker**: Consider failing open if Redis is unavailable

### Performance Tuning

- **Lua Script Atomicity**: The sliding window uses atomic Lua scripts
- **Key Expiration**: Keys auto-expire after the window period
- **Memory**: Each key uses O(n) memory where n = requests in window

### Security

- **API Key Hashing**: In production, hash API keys before using as Redis keys
- **Rate Limit by Account**: Consider per-account limits for authenticated users
- **Distributed Denial of Service**: Rate limiting alone won't stop DDoS; use a CDN/WAF

## Mono Framework Integration

This demo showcases mono framework patterns:

- **Module Interface**: `ratelimit.Module` implements mono's module lifecycle
- **Dependency Injection**: API module receives rate limit module reference
- **Graceful Shutdown**: Proper cleanup of Redis connections on shutdown

## Dependencies

- [go-monolith/mono](https://github.com/go-monolith/mono) - Modular monolith framework
- [gofiber/fiber](https://github.com/gofiber/fiber) - Express-inspired web framework
- [redis/go-redis](https://github.com/redis/go-redis) - Redis client for Go
- [gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown) - Graceful shutdown handling

## License

MIT License - See LICENSE file for details.
