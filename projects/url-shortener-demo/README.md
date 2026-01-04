# URL Shortener Demo

A URL shortening service built with the Mono Framework, demonstrating the `kv-jetstream` plugin and event-driven architecture patterns.

## Features

- **URL Shortening**: Generate short codes for long URLs
- **URL Redirection**: Redirect short codes to original URLs
- **Statistics Tracking**: Track access counts and analytics
- **Event-Driven Analytics**: Real-time event publishing and consumption
- **TTL Support**: Optional expiration for short URLs

## Why Use kv-jetstream for URL Mappings?

The `kv-jetstream` plugin is ideal for URL shortening because:

### 1. Fast Key-Value Lookups

URL shortening requires extremely fast lookups—users expect instant redirects. `kv-jetstream` provides:

- **Sub-millisecond reads**: Backed by NATS JetStream's optimized KV store
- **In-memory option**: Use `MemoryStorage` for fastest access
- **Simple API**: Direct key→value mapping perfect for shortcode→URL

```go
// Fast lookup by short code
data, err := bucket.Get(shortCode)
```

### 2. Built-in TTL Support

Many URL shorteners need expiring links. `kv-jetstream` handles this natively:

```go
// Create URL that expires in 24 hours
bucket.Set(shortCode, data, 24*time.Hour)
```

No external cleanup jobs needed—expired keys are automatically purged.

### 3. Optimistic Locking for Counters

Access counters need atomic updates. `kv-jetstream` provides revision-based locking:

```go
// Get current value with revision
entry, _ := bucket.GetEntry(shortCode)

// Update with revision check
_, err := bucket.Update(shortCode, newData, 0, entry.Revision)
if errors.Is(err, kvjetstream.ErrRevisionMismatch) {
    // Concurrent modification detected, retry
}
```

### 4. No External Dependencies

Unlike Redis or other KV stores, `kv-jetstream` runs embedded:

- **Zero configuration**: No separate database to manage
- **Single binary deployment**: Everything runs in one process
- **Consistent behavior**: Same API whether testing or in production

### When NOT to Use kv-jetstream

Consider alternatives if you need:

- **Persistence across restarts**: Use `FileStorage` or an external database
- **Multi-region replication**: Use a distributed database
- **Very large datasets (>1GB)**: Consider Redis or dedicated storage

## How UsePluginModule Interface Works

The `UsePluginModule` interface enables dependency injection for plugins:

### Registration Flow

```go
// 1. Create and register the plugin in main.go
kvStore, _ := kvjetstream.New(kvjetstream.Config{
    Buckets: []kvjetstream.BucketConfig{
        {Name: "urls", Storage: kvjetstream.MemoryStorage},
    },
})
app.RegisterPlugin(kvStore, "kv")  // "kv" is the alias

// 2. Module implements UsePluginModule interface
type Module struct {
    kv     *kvjetstream.PluginModule
    bucket kvjetstream.KVStoragePort
}

var _ mono.UsePluginModule = (*Module)(nil)

// 3. Framework calls SetPlugin BEFORE Start()
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
    if alias == "kv" {
        m.kv = plugin.(*kvjetstream.PluginModule)
    }
}

// 4. Module accesses bucket in Start()
func (m *Module) Start(ctx context.Context) error {
    m.bucket = m.kv.Bucket("urls")
    return nil
}
```

### Why This Pattern?

1. **Explicit dependencies**: Modules declare what plugins they need
2. **Type safety**: Type assertion ensures correct plugin type
3. **Lifecycle order**: Framework ensures plugins start before modules
4. **Testability**: Easy to mock plugins in tests

### Multiple Plugins

Modules can receive multiple plugins by alias:

```go
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
    switch alias {
    case "cache":
        m.cache = plugin.(*kvjetstream.PluginModule)
    case "storage":
        m.storage = plugin.(*fsjetstream.PluginModule)
    }
}
```

## Event-Driven Analytics Pattern

This demo shows event-driven communication between modules:

### Event Emitter (Shortener Module)

```go
// Declare events
var URLCreatedV1 = helper.EventDefinition[URLCreatedEvent](
    "shortener", "URLCreated", "v1",
)

// Implement EventEmitterModule
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        URLCreatedV1.ToBase(),
    }
}

// Publish events
URLCreatedV1.Publish(m.eventBus, event, nil)
```

### Event Consumer (Analytics Module)

```go
// Implement EventConsumerModule
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
    def, _ := registry.GetEventByName("URLCreated", "v1", "shortener")
    return registry.RegisterEventConsumer(def, m.handleURLCreated, m)
}

func (m *Module) handleURLCreated(ctx context.Context, msg *mono.Msg) error {
    var event URLCreatedEvent
    json.Unmarshal(msg.Data, &event)
    // Process event...
    return nil
}
```

### Benefits

- **Loose coupling**: Analytics doesn't depend on shortener module
- **Scalability**: Add more consumers without changing emitter
- **Observability**: Events provide audit trail

## API Endpoints

### URL Shortening

```bash
# Shorten a URL
POST /api/v1/shorten
Content-Type: application/json
{"url": "https://example.com/very-long-path", "ttl_seconds": 3600}

# Response
{"short_code": "abc123XY", "short_url": "http://localhost:8080/abc123XY", ...}
```

### URL Redirection

```bash
# Redirect (returns 307 Temporary Redirect)
GET /:shortCode
```

### Statistics

```bash
# Get stats for a URL
GET /api/v1/stats/:shortCode

# List all URLs
GET /api/v1/urls

# Delete a URL
DELETE /api/v1/urls/:shortCode
```

### Analytics

```bash
# Get analytics summary
GET /api/v1/analytics

# Get recent access logs
GET /api/v1/analytics/logs?limit=100
```

## Project Structure

```
url-shortener-demo/
├── main.go                           # Application entry point
├── go.mod
├── demo.sh                           # Demo script
├── README.md
└── modules/
    ├── shortener/                    # URL shortening service
    │   ├── module.go                 # UsePluginModule, EventEmitterModule
    │   ├── service.go                # Business logic
    │   ├── types.go                  # Data types
    │   ├── errors.go                 # Sentinel errors
    │   └── events.go                 # Event definitions
    ├── analytics/                    # Event consumer for analytics
    │   ├── module.go                 # EventConsumerModule
    │   └── types.go                  # Analytics store
    └── httpserver/                   # HTTP API with Fiber
        ├── module.go                 # HTTP server lifecycle
        └── handlers.go               # Request handlers
```

## Running the Demo

### Start the Server

```bash
go run .
```

### Run the Demo Script

```bash
./demo.sh
```

The demo script will:
1. Shorten a URL
2. Access it (redirect)
3. View statistics
4. Show analytics
5. Delete the URL

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `BASE_URL` | `http://localhost:8080` | Base URL for short links |
| `JETSTREAM_DIR` | `/tmp/url-shortener-demo` | JetStream storage directory |

## Key Concepts Demonstrated

1. **UsePluginModule**: Receiving kv-jetstream plugin via dependency injection
2. **EventEmitterModule**: Publishing domain events (URLCreated, URLAccessed)
3. **EventConsumerModule**: Consuming events for analytics tracking
4. **Fiber HTTP Framework**: REST API with middleware
5. **Sentinel Errors**: Proper error handling with errors.Is()
6. **Optimistic Locking**: Safe concurrent counter updates
