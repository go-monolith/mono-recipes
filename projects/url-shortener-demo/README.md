# URL Shortener Demo

A URL shortening service built with the Mono Framework (mono v0.1.1), demonstrating the `kv-jetstream` plugin, event-driven architecture and module boundaries (typed contracts, consumer-owned ports and adapters, strict mode).

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
    kv *kvjetstream.PluginModule
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
    bucket := m.kv.Bucket("urls")
    service, err := NewService(bucket, m.baseURL)
    m.service = service
    return err
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

This demo shows event-driven communication between modules. The events are
declared once, in the **emitter's** contract package, and the consumer imports
that package (never the shortener module itself).

### Event Emitter (Shortener Module)

```go
// modules/shortener/contract/contract.go
var URLCreatedV1 = helper.EventDefinition[URLCreatedEvent](
    "shortener", "URLCreated", "v1",
)

// modules/shortener/module.go: implement EventEmitterModule
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        contract.URLCreatedV1.ToBase(),
        contract.URLAccessedV1.ToBase(),
    }
}

// Publish events
contract.URLCreatedV1.Publish(m.eventBus, event, nil)
```

### Event Consumer (Analytics Module)

```go
// modules/analytics/module.go: implement EventConsumerModule
import shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"

func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
    return helper.RegisterTypedEventConsumer(registry,
        shortenercontract.URLCreatedV1, m.handleURLCreated, m)
}

// The payload arrives already decoded.
func (m *Module) handleURLCreated(ctx context.Context, event shortenercontract.URLCreatedEvent, _ *mono.Msg) error {
    m.store.RecordURLCreated(event.ShortCode, event.OriginalURL, event.CreatedAt)
    return nil
}
```

### Benefits

- **Loose coupling**: Analytics doesn't depend on the shortener module, only on its contract
- **No name strings**: no `GetEventByName("URLCreated", "v1", "shortener")` lookups to get wrong
- **Scalability**: Add more consumers without changing emitter
- **Observability**: Events provide audit trail

## Module Boundaries

Modules talk to each other only through **typed contracts** published in a
`contract` sub-package. `http-server` is the only module that calls services;
it owns its ports and adapters.

```
shortener/contract   ShortenURL, ResolveURL, GetStats, ListURLs, DeleteURL   (services.shortener.*)
                     URLCreatedV1, URLAccessedV1                             (events.shortener.v1.*)
                     ErrURLNotFound, ErrInvalidURL, ErrInvalidShortCode, ErrURLExpired
analytics/contract   GetSummary, GetLogs                                     (services.analytics.*)
```

**Provider** (`shortener/module.go`) registers through its own contract, with a typed handler:

```go
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    if err := contract.ShortenURL.Register(container, m.handleShortenURL); err != nil {
        return err
    }
    // ... ResolveURL, GetStats, ListURLs, DeleteURL
}

func (m *Module) handleShortenURL(ctx context.Context, req contract.ShortenRequest, _ *mono.Msg) (contract.ShortenResponse, error)
```

**Consumer** (`httpserver/`): `ports.go` declares what the handlers need
(`ShortenerPort`, `AnalyticsPort`, in the providers' contract DTOs), and
`adapter.go` implements them over the contracts with the container received
in `SetDependencyServiceContainer`:

```go
type shortenerAdapter struct{ container mono.ServiceContainer }

func (a shortenerAdapter) GetStats(ctx context.Context, shortCode string) (shortenercontract.StatsResponse, error) {
    resp, err := shortenercontract.GetStats.Call(ctx, a.container, shortenercontract.ShortCodeRequest{ShortCode: shortCode})
    return resp, mapShortenerError(err)
}

// ConsumeServices lists every contract the module calls; the framework
// validates it at startup against the declared dependencies.
func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
    return []types.BaseServiceDefinition{
        shortenercontract.ShortenURL.ToBase(), /* ... */ analyticscontract.GetLogs.ToBase(),
    }
}
```

A shortener handler reports a domain error by returning it (for example
`contract.ErrURLNotFound`). It travels back as a `RemoteError` carrying the
message (the framework also logs it as a handler error), and
`mapShortenerError` turns it back into the contract's sentinel so the HTTP
handlers can use `errors.Is` to choose 404/400/410.

`main.go` runs with `mono.WithStrictModuleBoundaries()`: a module that calls a
service of a module it did not declare as a dependency, or registers on a
dependency's container, fails instead of only being reported at `Stop`.

### Conformance tests

```bash
go test ./...
```

- `modules/httpserver/module_test.go` and `modules/analytics/module_test.go` run
  `monotest.CheckModule` (no NATS): stand-in containers register the providers'
  contracts, and the exercise calls every provided and consumed contract.
- `modules/shortener/module_test.go` is a real-app test (the kv-jetstream
  plugin needs JetStream, which monotest does not emulate): default (non-strict) mode so violations are recorded, every
  shortener contract called, both events received through the contract
  definitions, and `app.ModuleBoundaries()` without violations.

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
├── main.go                           # Application entry point (strict module boundaries)
├── go.mod
├── demo.sh                           # Demo script
├── README.md
└── modules/
    ├── shortener/                    # URL shortening service
    │   ├── contract/
    │   │   └── contract.go           # Published port: DTOs, service contracts, events, errors
    │   ├── module.go                 # UsePluginModule, EventEmitterModule, RegisterServices
    │   ├── service.go                # Business logic (kv-jetstream bucket)
    │   ├── service_test.go           # Validation unit tests
    │   └── module_test.go            # Real-app boundary test (JetStream)
    ├── analytics/                    # Event consumer for analytics
    │   ├── contract/
    │   │   └── contract.go           # Published port: summary/logs contracts
    │   ├── module.go                 # EventConsumerModule, RegisterServices
    │   ├── types.go                  # Analytics store
    │   ├── types_test.go             # Store unit tests
    │   └── module_test.go            # monotest.CheckModule conformance test
    └── httpserver/                   # HTTP API with Fiber
        ├── ports.go                  # Driven ports: ShortenerPort, AnalyticsPort
        ├── adapter.go                # Adapters over the shortener/analytics contracts
        ├── module.go                 # HTTP server lifecycle, Dependencies, ConsumeServices
        ├── handlers.go               # Request handlers
        └── module_test.go            # monotest.CheckModule conformance test
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
3. **EventConsumerModule**: Consuming events through the emitter's contract definitions
4. **Typed service contracts**: `contract` packages, consumer-owned ports and adapters, `ConsumeServices()`
5. **Strict module boundaries**: `mono.WithStrictModuleBoundaries()` and `monotest.CheckModule`
6. **Fiber HTTP Framework**: REST API with middleware
7. **Sentinel Errors**: Contract errors mapped back from remote errors, checked with errors.Is()
8. **Optimistic Locking**: Safe concurrent counter updates
