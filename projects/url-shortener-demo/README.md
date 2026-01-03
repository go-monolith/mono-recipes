# URL Shortener Demo

A URL shortening service built with the Mono framework, demonstrating **Fiber HTTP framework** integration and **NATS JetStream KV Store** for persistent key-value storage.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Clients                              │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API Module (Fiber)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ POST /shorten│  │GET /:code   │  │ GET /stats/:code        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────┬───────────────────────────────────┘
                              │ DependentModule
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Shortener Module (ServiceProviderModule)            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Service Layer: ShortenURL, ResolveURL, GetStats, RecordAccess││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Storage Layer: JetStream KV Store (urls, url-stats buckets)││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  EventEmitterModule: Publishes URLCreated, URLAccessed events││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────────┘
                              │ Events via NATS
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│            Analytics Module (EventConsumerModule)                │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Consumes: URLCreated, URLAccessed events                    ││
│  │  Logs analytics data for monitoring and insights             ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    NATS JetStream                                │
│  ┌────────────────┐  ┌────────────────┐  ┌───────────────────┐  │
│  │ KV: urls       │  │ KV: url-stats  │  │ Stream: events    │  │
│  │ (URL mappings) │  │ (access stats) │  │ (analytics)       │  │
│  └────────────────┘  └────────────────┘  └───────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Why Use JetStream KV for URL Mappings?

### Perfect Fit for URL Shortening

1. **Simple Key-Value Model**: URL shortening is fundamentally a key-value problem:
   - Key: Short code (e.g., `abc123`)
   - Value: Original URL and metadata

2. **Built-in TTL Support**: JetStream KV natively supports per-key TTL:
   ```go
   _, err := store.urlBucket.Put(ctx, shortCode, data, jetstream.WithTTL(ttl))
   ```
   - URLs can automatically expire without cleanup jobs
   - Configurable per-URL expiration

3. **Atomic Operations**: KV store provides atomic get/put operations:
   - No race conditions on URL creation
   - Consistent reads for high-traffic redirects

4. **High Performance**: Optimized for fast lookups:
   - Sub-millisecond read latency for redirects
   - Efficient for the read-heavy workload of URL shortening

5. **Persistence with Replication**: JetStream provides:
   - Persistent storage across restarts
   - Optional replication for high availability
   - File-based storage for durability

### Comparison with Alternatives

| Feature | JetStream KV | Redis | PostgreSQL |
|---------|-------------|-------|------------|
| TTL Support | Native | Native | Requires triggers |
| Persistence | Built-in | Optional | Built-in |
| Messaging | Integrated | Pub/Sub only | Requires LISTEN/NOTIFY |
| Operational Complexity | Single system | Separate system | Separate system |
| Latency | Sub-ms | Sub-ms | 1-5ms |

## Event-Driven Analytics Pattern

This demo showcases the **EventEmitterModule** and **EventConsumerModule** pattern for decoupled analytics:

### Event Flow

```
[URL Created] ──► URLCreatedEvent ──► Analytics Module
                                            │
                                            ▼
                                      Log/Store metrics

[URL Accessed] ──► URLAccessedEvent ──► Analytics Module
                                             │
                                             ▼
                                       Log/Store metrics
```

### Benefits of Event-Driven Analytics

1. **Decoupled Architecture**
   - Core shortener logic doesn't know about analytics
   - Analytics module can be added/removed without code changes
   - Easy to add more consumers (e.g., fraud detection, rate limiting)

2. **Non-Blocking Operations**
   - URL redirects are not slowed by analytics processing
   - Events are published asynchronously
   - Analytics failures don't affect core functionality

3. **Scalability**
   - Analytics processing can scale independently
   - Multiple analytics consumers can process events in parallel
   - Event replay capability for reprocessing

### Event Definitions

```go
// URLCreated event - emitted when a new short URL is created
var URLCreatedV1 = helper.EventDefinition[URLCreatedEvent](
    "url",        // domain
    "URLCreated", // event name
    "v1",         // version
)

// URLAccessed event - emitted when a short URL is accessed
var URLAccessedV1 = helper.EventDefinition[URLAccessedEvent](
    "url",         // domain
    "URLAccessed", // event name
    "v1",          // version
)
```

## Scalability and TTL Considerations

### Horizontal Scaling

1. **Stateless API Layer**: Fiber HTTP servers can be scaled horizontally
   - Load balancer distributes requests
   - No session state stored in application

2. **Shared State via JetStream**: All instances share the same KV store
   - Consistent URL resolution across instances
   - No sticky sessions required

3. **Event Processing**: Analytics consumers can scale independently
   - Queue groups for load balancing across consumers
   - Durable subscriptions for reliability

### TTL Strategies

1. **Default TTL**: URLs without explicit TTL use bucket-level default
   ```go
   jetstream.KeyValueConfig{
       Bucket:  "urls",
       TTL:     24 * time.Hour * 365, // 1 year default
   }
   ```

2. **Custom TTL**: Per-URL expiration for premium features
   ```json
   {
     "url": "https://example.com/long-path",
     "ttl_seconds": 3600
   }
   ```

3. **Statistics Bucket**: Separate TTL for stats data
   ```go
   jetstream.KeyValueConfig{
       Bucket:  "url-stats",
       TTL:     24 * time.Hour * 30, // 30 days for stats
   }
   ```

### Performance Considerations

1. **Short Code Generation**: Base62 encoding provides:
   - 62^6 = 56+ billion unique codes with 6 characters
   - Collision checking with retry logic

2. **Access Counting**: Atomic increment operations
   ```go
   stats.AccessCount++
   stats.LastAccess = &now
   ```

3. **Redirect Latency**: Optimized for fast redirects
   - Single KV lookup for URL resolution
   - Async event publishing (non-blocking)
   - HTTP 301 redirect for browser caching

## Mono Framework Patterns Demonstrated

### 1. ServiceProviderModule
The shortener module exposes services via request-reply:
```go
helper.RegisterTypedRequestReplyService(
    registry,
    ServiceShortenURL,
    m.handleShortenURL,
)
```

### 2. EventEmitterModule
Publishing events for analytics:
```go
func (m *ShortenerModule) RegisterEventEmitters(registry mono.EventRegistry) error {
    return registry.RegisterEmitter(events.URLCreatedV1, events.URLAccessedV1)
}
```

### 3. EventConsumerModule
Consuming events for analytics processing:
```go
func (m *AnalyticsModule) RegisterEventConsumers(registry mono.EventRegistry) error {
    return helper.RegisterTypedEventConsumer(
        registry, events.URLCreatedV1, m.handleURLCreated, m,
    )
}
```

### 4. DependentModule
API module depends on shortener module:
```go
func (m *APIModule) Dependencies() []string {
    return []string{"shortener"}
}
```

### 5. HealthCheckableModule
All modules implement health checks:
```go
func (m *ShortenerModule) Health(ctx context.Context) mono.HealthStatus {
    return mono.HealthStatus{
        Healthy: m.store != nil,
        Message: "operational",
    }
}
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/shorten` | Create a shortened URL |
| GET | `/:code` | Redirect to original URL |
| GET | `/api/v1/stats/:code` | Get URL statistics |
| GET | `/health` | Health check |

### Create Short URL

```bash
curl -X POST http://localhost:3000/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/path"}'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "short_code": "abc123",
  "short_url": "http://localhost:3000/abc123",
  "original_url": "https://example.com/very/long/path",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Create with Custom Code and TTL

```bash
curl -X POST http://localhost:3000/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "custom_code": "mylink",
    "ttl_seconds": 86400
  }'
```

### Get Statistics

```bash
curl http://localhost:3000/api/v1/stats/abc123
```

Response:
```json
{
  "short_code": "abc123",
  "short_url": "http://localhost:3000/abc123",
  "original_url": "https://example.com/very/long/path",
  "access_count": 42,
  "created_at": "2024-01-15T10:30:00Z",
  "last_access": "2024-01-15T12:45:00Z"
}
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose

### Running the Demo

1. Start NATS JetStream:
   ```bash
   docker-compose up -d
   ```

2. Run the application:
   ```bash
   go run main.go
   ```

3. Test the API:
   ```bash
   ./demo.sh
   ```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `PORT` | `3000` | HTTP server port |
| `BASE_URL` | `http://localhost:3000` | Base URL for short links |

## Project Structure

```
url-shortener-demo/
├── domain/
│   └── url/
│       └── entity.go          # Domain entities
├── events/
│   └── url_events.go          # Event definitions
├── modules/
│   ├── shortener/
│   │   ├── storage.go         # JetStream KV store
│   │   ├── codegen.go         # Short code generator
│   │   ├── service.go         # Business logic
│   │   ├── module.go          # ServiceProviderModule
│   │   ├── adapter.go         # Cross-module adapter
│   │   └── types.go           # Request/response types
│   ├── analytics/
│   │   └── module.go          # EventConsumerModule
│   └── api/
│       ├── module.go          # Fiber HTTP module
│       ├── handlers.go        # HTTP handlers
│       └── types.go           # API types
├── main.go                    # Entry point
├── docker-compose.yml         # NATS JetStream
├── demo.sh                    # Demo script
└── README.md
```

## License

MIT License - See LICENSE file for details.
