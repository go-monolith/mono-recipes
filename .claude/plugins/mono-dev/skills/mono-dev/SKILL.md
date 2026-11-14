---
name: mono-dev
description: This skill should be used when the user asks to "create a Mono application", "add a new Mono module", "create a Mono plugin", "create a Mono middleware", "use kv-jetstream plugin", "use fs-jetstream plugin", "create Python client to connect to Mono app", "create Node.js client to connect to Mono app", "add polyglot client for Mono app", "add background jobs in Mono app", "add queue-group workers", "define a service contract", "add a contract package", "check module boundaries", "enforce hexagonal architecture", "ports and adapters in Mono", "test a Mono module without NATS", "add a cron job in Mono app", "schedule a recurring task", "add a scheduled service", or mentions "mono framework", "MonoApplication", "ServiceContainer", "ChannelService", "RequestReplyService", "QueueGroupService", "StreamConsumerService", "CronService", "RegisterCronService", "CronServiceConfig", "EventBus", "EventRegistry", "EventEmitter", "EventConsumer", "EventStreamConsumer", "PluginModule", "UsePluginModule", "MiddlewareModule", "OnServiceRegistration", "OnModuleLifecycle", "OnOutgoingMessage", "kv-jetstream", "fs-jetstream", "KVStoragePort", "FileStoragePort", "QGHP", "ServiceConsumerModule", "ConsumeServices", "helper.RequestReplyService", "helper.QueueGroupService", "helper.StreamConsumerService", "BaseServiceDefinition", "WithStrictModuleBoundaries", "ModuleBoundaries", "ErrModuleBoundaryViolation", "monotest", "CheckModule", "nats.py", "nats.js". Provides best practices for developing modular monolith applications with the go-monolith/mono framework.
version: 0.6.1
---

# Mono Framework Development

Build modular monolith applications with the go-monolith/mono framework. This skill provides best practices for creating modules, services, and event-driven communication.

## Framework Overview

The Mono Framework enables modular monolith architecture with embedded NATS.io messaging. Core components:

- **MonoApplication**: Main entry point for initialization and lifecycle
- **Module**: Independent component with Start/Stop lifecycle
- **ServiceContainer**: Per-module registry for service registration and discovery
- **EventBus**: Publish/subscribe messaging for loose coupling
- **EventRegistry**: Event discovery and consumer registration

## Quick Start

### Creating a MonoApplication

```go
import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    gfshutdown "github.com/gelmium/graceful-shutdown"
    "github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
    app, err := mono.NewMonoApplication(
        mono.WithShutdownTimeout(shutdownTimeout),
        mono.WithLogLevel(mono.LogLevelInfo),
        mono.WithLogFormat(mono.LogFormatText),
        // Optional: for persistent JetStream storage
        // mono.WithJetStreamStorageDir("/tmp/my-app"),
        // Optional: custom NATS port
        // mono.WithNATSPort(4222),
    )
    if err != nil {
        log.Fatalf("Failed to create app: %v", err)
    }

    // Register modules
    app.Register(&MyModule{})

    // Start
    ctx := context.Background()
    if err := app.Start(ctx); err != nil {
        log.Fatalf("Failed to start: %v", err)
    }

    log.Println("Application started successfully")

    // Graceful shutdown (standard pattern)
    wait := gfshutdown.GracefulShutdown(
        context.Background(),
        shutdownTimeout,
        map[string]gfshutdown.Operation{
            "mono-app": func(ctx context.Context) error {
                return app.Stop(ctx)
            },
        },
    )

    exitCode := <-wait
    os.Exit(exitCode)
}
```

### Creating a Basic Module

Every module implements three methods:

```go
type MyModule struct{}

// Compile-time interface check
var _ mono.Module = (*MyModule)(nil)

func (m *MyModule) Name() string { return "my-module" }

func (m *MyModule) Start(ctx context.Context) error {
    slog.Info("Starting my-module")
    return nil
}

func (m *MyModule) Stop(ctx context.Context) error {
    slog.Info("Stopping my-module")
    return nil
}
```

## Module Interface Hierarchy

Modules can implement optional interfaces for additional capabilities:

| Interface | Purpose |
|-----------|---------|
| `EventBusAwareModule` | Receive EventBus for publishing |
| `ServiceProviderModule` | Register services for other modules |
| `DependentModule` | Declare dependencies on other modules |
| `SetDependencyServiceContainer` | Receive dependency service containers |
| `ServiceConsumerModule` | Declare consumed service contracts, validated at startup (`pkg/types`, v0.1.0+) |
| `EventEmitterModule` | Declare events the module emits |
| `EventConsumerModule` | Register event consumer handlers |
| `HealthCheckableModule` | Report custom health status |
| `PluginModule` | Start first, stop last (cross-cutting) |
| `UsePluginModule` | Receive plugin instances |
| `MiddlewareModule` | Intercept and wrap service handlers |

## Service Communication Patterns

Five patterns for inter-module and scheduled communication:

| Type | Use Case | Latency |
|------|----------|---------|
| **Channel** | In-process, high-throughput | ~microseconds |
| **Request-Reply** | Synchronous with response | ~1ms |
| **Queue Group** | Async, load-balanced | ~1ms |
| **Stream Consumer** | Durable, at-least-once | ~5ms |
| **Cron** | Server-side scheduled (cron/interval) | per schedule |

### Registering RequestReplyService

```go
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterRequestReplyService(
        "process-payment",
        m.handleProcessPayment,
    )
}

func (m *PaymentModule) handleProcessPayment(
    ctx context.Context, msg *mono.Msg) ([]byte, error) {
    var req PaymentRequest
    json.Unmarshal(msg.Data, &req)
    // Process and return response
    return json.Marshal(response)
}
```

### Registering QueueGroupService (Fire-and-Forget)

Use `mono.QGHP` (Queue Group Handler Params) for background job processing:

```go
func (m *WorkerModule) RegisterServices(container mono.ServiceContainer) error {
    // Single queue group
    return container.RegisterQueueGroupService(
        "process-job",
        mono.QGHP{
            QueueGroup: "job-workers",
            Handler:    m.handleJob,
        },
    )
}

// Multiple handlers on same service (load balanced)
func (m *WorkerModule) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterQueueGroupService(
        "process-job",
        mono.QGHP{QueueGroup: "worker-1", Handler: m.handleJob1},
        mono.QGHP{QueueGroup: "worker-2", Handler: m.handleJob2},
    )
}

func (m *WorkerModule) handleJob(ctx context.Context, msg *mono.Msg) error {
    var job JobRequest
    json.Unmarshal(msg.Data, &job)
    // Process job (fire-and-forget, no response)
    return nil
}
```

### Registering CronService (Scheduled)

Run a handler on a server-side schedule. The schedule lives in the embedded NATS
JetStream message scheduler, so in a multi-node deployment **exactly one**
occurrence fires per tick — no client-side ticker, no leader election. Delivery
is at-least-once with explicit ack (return `nil` to Ack, an error to Nak).

CronService requires JetStream — create the app with a storage dir:
`mono.NewMonoApplication(mono.WithJetStreamStorageDir("/tmp/my-app"))`.

```go
func (m *MaintenanceModule) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterCronService(
        "cleanup",
        mono.CronServiceConfig{
            Schedule: "0 3 * * *",                     // schedule (required)
            Payload:  []byte(`{"event":"cleanup"}`),   // static body per tick
            TimeZone: "UTC",                            // optional; default UTC
            TTL:      1 * time.Hour,                    // optional; drop stale ticks
            // SourceSubject: "metrics.raw",            // deliver last msg on a
            //   subject instead of Payload (downsampling); exclusive with Payload
            // Deprecated: true,                        // purge schedule + stop
            //   consumer, keep the registration (phase one of safe retirement)
        },
        m.handleCleanup,
    )
}

func (m *MaintenanceModule) handleCleanup(ctx context.Context, msg *mono.Msg) error {
    // Runs on each occurrence; msg.Data carries the configured Payload.
    return m.purgeExpiredRecords(ctx)
}
```

**Schedule formats** (`CronServiceConfig.Schedule`):

| Form | Example | Notes |
|------|---------|-------|
| Interval | `@every 30s` | minimum `1s`; no time zone |
| Five-field cron | `0 3 * * *` | normalized to six-field by prepending `0` seconds |
| Six-field cron | `0 */5 * * * *` | seconds-first (`sec min hour dom mon dow`) |
| Named alias | `@daily` | also `@hourly`, `@weekly`, `@monthly`, `@yearly` |

> Both five-field and six-field cron expressions are accepted (mono v0.0.8+).
> Use six fields for sub-minute precision. See `examples/cron-service.go`.

### Typed Service Contracts (Recommended, v0.1.0+)

Declare each service once in the provider's `contract` sub-package. The provider registers through it and consumers call through it, so the name, types and encoding are written once. See **Module Boundaries** below.

```go
// payment/contract/contract.go
var Process = helper.RequestReplyService[PaymentRequest, PaymentResponse]("payment", "process-payment")

// payment/module.go (provider)
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
    return contract.Process.Register(container, m.process) // typed handler
}

// order/adapter.go (consumer)
resp, err := paymentcontract.Process.Call(ctx, m.paymentContainer, paymentcontract.PaymentRequest{Amount: 100})
```

`helper.QueueGroupService[T]` (`Register`/`Send`) and `helper.StreamConsumerService[T]` (`Register`/`Publish`) follow the same shape.

### Typed Handlers without a Contract

The lower-level helpers still work, for example for a service only polyglot clients call:

```go
import "github.com/go-monolith/mono/pkg/helper"

func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    return helper.RegisterTypedRequestReplyService(
        container,
        "calculate",
        json.Unmarshal,  // Request unmarshaler
        json.Marshal,    // Response marshaler
        m.handleCalculate,
    )
}

// Handler with typed request/response
func (m *Module) handleCalculate(
    ctx context.Context,
    req CalculateRequest,
    msg *mono.Msg,
) (CalculateResponse, error) {
    // Process typed request, return typed response
    return CalculateResponse{Result: req.A + req.B}, nil
}
```

### Consuming Services (Consumer)

Prefer a contract call (above). The string-based helper below is the lower-level form:

```go
func (m *OrderModule) Dependencies() []string {
    return []string{"payment"}
}

func (m *OrderModule) SetDependencyServiceContainer(
    dep string, container mono.ServiceContainer) {
    if dep == "payment" {
        m.paymentContainer = container
    }
}

func (m *OrderModule) processOrder(ctx context.Context) error {
    // Type-safe service call
    var response PaymentResponse
    err := helper.CallRequestReplyService(
        ctx,
        m.paymentContainer,
        "process-payment",
        json.Marshal,
        json.Unmarshal,
        &PaymentRequest{Amount: 100},
        &response,
    )
    return err
}
```

## Module Boundaries (Ports and Adapters, v0.1.0+)

The framework can check that modules follow the hexagonal pattern. Full guide: **`references/module-boundaries.md`**. Working code: **`examples/module-boundaries/`**.

**Recommended path:**

1. Give each module a `contract` sub-package: DTOs, typed service definitions, emitted events. It is the only package of the module others import, and it imports only the standard library, `mono/pkg/types` and `mono/pkg/helper`.
2. Register and call through contracts. No service-name strings in module code.
3. In consumers, put driven ports in `ports.go` and adapters in `adapter.go`.
4. Implement `ConsumeServices()` so startup fails fast on a missing or mistyped service.
5. Add one `monotest.CheckModule` test per module.
6. Run in the default mode and read `app.ModuleBoundaries()` or the `Stop` log.
7. Fix what it reports, then enable `mono.WithStrictModuleBoundaries()`.

```go
import "github.com/go-monolith/mono/pkg/types" // no mono.* alias yet

// Validated when the module starts: the provider must be a declared
// dependency (or a plugin), have registered the service, with the same kind.
func (m *OrderModule) ConsumeServices() []types.BaseServiceDefinition {
    return []types.BaseServiceDefinition{
        inventorycontract.CheckStock.ToBase(),
        paymentcontract.Process.ToBase(),
    }
}
```

**At runtime** the container from `SetDependencyServiceContainer` is a read-only view, and the bus from `SetEventBus` is scoped to the module. Four violations are detected:

| Kind | Cause |
|------|-------|
| `register-on-dependency` | Mutating a dependency's container |
| `channel-consumer-mismatch` | `GetChannelService` under another module's name |
| `undeclared-service-subject` | Raw publish/request to `services.<undeclared-module>.*` |
| `foreign-service-subscription` | Subscribing to another module's service subject |

By default they are warned once and listed in `app.ModuleBoundaries()`. With `mono.WithStrictModuleBoundaries()` they fail with `ErrModuleBoundaryViolation`.

**In tests**, `monotest` runs a module without NATS:

```go
func TestOrderModuleConformance(t *testing.T) {
    inventory := monotest.NewContainer("inventory", monotest.NewEventBus())
    // register stand-in services via the provider's contract or monotest.StubRequestReply

    monotest.CheckModule(t, order.NewModule(),
        monotest.WithDependency("inventory", inventory),
        monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
            _, err := contract.PlaceOrder.Call(ctx, h.Container(), contract.PlaceOrderRequest{ProductID: "laptop", Quantity: 1})
            return err
        }))
}
```

Not checked: imports between module packages, modules handed each other's objects directly, `app.Services(name)` (host code only), and JetStream publishes.

## Event-Driven Communication

Events enable loose coupling - consumers do NOT declare dependencies on emitters.

### Defining Events

```go
import "github.com/go-monolith/mono/pkg/helper"

var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEvent](
    "order",        // Module name
    "OrderCreated", // Event name
    "v1",           // Version
)

type OrderCreatedEvent struct {
    OrderID    string  `json:"order_id"`
    CustomerID string  `json:"customer_id"`
    Amount     float64 `json:"amount"`
}
```

### Emitting Events

```go
func (m *OrderModule) SetEventBus(bus mono.EventBus) {
    m.eventBus = bus
}

func (m *OrderModule) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        OrderCreatedV1.ToBase(),
    }
}

func (m *OrderModule) createOrder(ctx context.Context) error {
    // Fire-and-forget
    return OrderCreatedV1.Publish(m.eventBus, event, nil)

    // Or durable (JetStream)
    ack, err := OrderCreatedV1.EventStreamPublish(ctx, m.eventBus, event, nil)
}
```

### Consuming Events

```go
func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    eventDef, ok := registry.GetEventByName("OrderCreated", "v1", "order")
    if !ok {
        return fmt.Errorf("event not found")
    }
    return registry.RegisterEventConsumer(eventDef, m.handleOrderCreated, m)
}

func (m *NotificationModule) handleOrderCreated(
    ctx context.Context, msg *mono.Msg) error {
    var event OrderCreatedEvent
    json.Unmarshal(msg.Data, &event)
    // Process event
    return nil
}
```

## Project Structure

Recommended layout for Mono applications:

```
my-app/
├── main.go                    # Application entry point
├── go.mod / go.sum            # Dependencies
├── README.md                  # Documentation
├── bin/                       # Compiled binaries
│   └── my-app
├── modules/                   # Domain modules
│   ├── api/                   # HTTP server module
│   │   ├── module.go
│   │   ├── handlers.go
│   │   └── routes.go
│   ├── user/                  # Business module with database (SOLID principles)
│   │   ├── contract/          # Published port: the only package other modules import
│   │   │   ├── contract.go    # DTOs + typed service contracts
│   │   │   └── events.go      # Events this module emits
│   │   ├── module.go          # Module implementation, ConsumeServices()
│   │   ├── module_test.go     # monotest.CheckModule conformance test
│   │   ├── ports.go           # Driven ports this module needs
│   │   ├── adapter.go         # Adapters over other modules' contracts
│   │   ├── service.go         # Service handlers
│   │   ├── service_test.go    # Unit tests
│   │   ├── types.go           # Internal domain types
│   │   ├── repository.go      # Repository interface
│   │   └── db/                # Module-owned database (sqlc)
│   │       ├── schema.sql     # Schema owned by this module
│   │       ├── sqlc.yaml      # sqlc config for this module
│   │       ├── queries/
│   │       │   └── users.sql
│   │       └── generated/     # sqlc generated code
│   │           ├── db.go
│   │           ├── models.go
│   │           └── query.sql.go
│   └── notification/          # Event consumer module
│       ├── module.go
│       └── handlers.go
├── domain/                    # Shared domain types (optional)
│   └── types.go
├── middleware/                # Custom middleware (optional)
│   └── ratelimit/
│       ├── middleware.go
│       └── limiter.go
└── client/                    # External clients (polyglot)
    ├── python/
    │   ├── client.py
    │   └── client_test.py
    └── nodejs/
        ├── client.js
        └── client.test.js
```

## Best Practices

### Do

- Keep modules focused on a single domain
- Use services for point-to-point calls (caller knows provider)
- Use events for broadcast/loose coupling (emitter doesn't know consumers)
- Declare dependencies explicitly via `DependentModule`
- Publish a `contract` sub-package per module; register and call services through typed contracts
- Declare consumed contracts with `ConsumeServices()` and add a `monotest.CheckModule` test per module
- Enable `mono.WithStrictModuleBoundaries()` once `app.ModuleBoundaries()` is clean
- Handle errors explicitly, wrap with context using `%w`
- Use structured logging with `log/slog` or pass in `app.Logger()` in module constructor
- Use compile-time interface checks: `var _ mono.Module = (*MyModule)(nil)`

### Don't

- Call module.Start() or module.Stop() directly (this is handled by framework)
- Define & call modules's function directly - use services or events
- Create circular dependencies - refactor or use events
- Import another module's package other than its `contract`
- Register on a dependency's container, or request another module's `services.*` subject on the event bus
- Store global state - use framework's dependency injection
- Panic in modules - return errors instead
- Share mutable state - communicate via messages

## Common Patterns

### Module with Health Check

```go
func (m *MyModule) Health(ctx context.Context) mono.HealthStatus {
    return mono.HealthStatus{
        Healthy: m.isConnected(),
        Status:  "Ready",
    }
}
```

### Using Plugins

```go
func (m *MyModule) SetPlugin(alias string, plugin mono.PluginModule) {
    if alias == "storage" {
        m.storage = plugin.(*fsjetstream.PluginModule)
    }
}
```

### Type-Safe Service Calls

```go
import "github.com/go-monolith/mono/pkg/helper"

err := helper.CallRequestReplyService(
    ctx,
    m.paymentContainer,
    "process-payment",
    json.Marshal,
    json.Unmarshal,
    &request,
    &response,
)
```

## Middleware System

Middleware modules intercept framework events and can observe or modify them. They start before regular modules and stop after, ensuring full coverage.

### Built-in Middleware

| Middleware | Purpose | Use Cases |
|------------|---------|-----------|
| **requestid** | Request ID tracking | Distributed tracing, log correlation |
| **accesslog** | Request/response logging | Performance monitoring, debugging |
| **audit** | Tamper-evident audit trail | Compliance, security auditing |

### Using Middleware

```go
import (
    "github.com/go-monolith/mono/middleware/requestid"
    "github.com/go-monolith/mono/middleware/accesslog"
)

// Register middleware BEFORE regular modules
requestIDMiddleware, _ := requestid.New()
app.Register(requestIDMiddleware)

logFile, _ := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
accessLogMiddleware, _ := accesslog.New(
    accesslog.WithOutput(logFile),
    accesslog.WithFormat(accesslog.FormatJSON),
)
app.Register(accessLogMiddleware)

// Then register regular modules
app.Register(&MyModule{})
```

### Creating Custom Middleware

Implement `MiddlewareModule` interface with hook methods:

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

## Plugin System

Plugins are specialized modules for cross-cutting infrastructure concerns. They start first and stop last, providing services like storage, caching, and external integrations.

### Built-in Plugins

| Plugin | Purpose | Use Cases |
|--------|---------|-----------|
| **kv-jetstream** | Key-value storage | Sessions, caching, config, distributed locks |
| **fs-jetstream** | File/object storage | Documents, media, uploads, large files |

### fs-jetstream Plugin

```go
import fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"

// In main.go
storage, err := fsjetstream.New(fsjetstream.Config{
    Buckets: []fsjetstream.BucketConfig{
        {
            Name:        "files",
            Description: "File storage bucket",
            MaxBytes:    1024 * 1024 * 1024, // 1GB
            Storage:     fsjetstream.FileStorage,
            Compression: true,
        },
    },
})
app.RegisterPlugin(storage, "storage")

// Module receives plugin via SetPlugin
func (m *MyModule) SetPlugin(alias string, plugin mono.PluginModule) {
    if alias == "storage" {
        m.storage = plugin.(*fsjetstream.PluginModule)
    }
}

// Access bucket in Start()
func (m *MyModule) Start(ctx context.Context) error {
    m.bucket = m.storage.Bucket("files")
    // bucket.Put(), bucket.Get(), bucket.Delete(), bucket.List()
    return nil
}
```

### kv-jetstream Plugin

```go
import kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"

kvStore, err := kvjetstream.New(kvjetstream.Config{
    Buckets: []kvjetstream.BucketConfig{
        {
            Name:    "sessions",
            Storage: kvjetstream.MemoryStorage,
            TTL:     24 * time.Hour,
        },
        {
            Name:    "config",
            Storage: kvjetstream.FileStorage,
        },
    },
})
app.RegisterPlugin(kvStore, "kv")

// Usage: bucket.Put(key, value), bucket.Get(key), bucket.Delete(key)
```

### Creating Custom Plugins

Implement `PluginModule` interface:

```go
type PluginModule interface {
    Module  // Name(), Start(), Stop()
    SetContainer(container ServiceContainer)
    Container() ServiceContainer
}
```

Define a public API (Port) for consumers and expose via `Port()` method.

## Additional Resources

### Reference Files

For detailed patterns and implementation examples:

- **`references/modules.md`** - Module interface patterns and lifecycle
- **`references/services.md`** - Service registration and consumption
- **`references/module-boundaries.md`** - Contracts, ports and adapters, `ConsumeServices`, boundary report, strict mode, `monotest`
- **`references/events.md`** - Event emitter and consumer patterns
- **`references/plugins.md`** - Plugin creation and built-in plugins
- **`references/middleware.md`** - Middleware hooks and built-in middleware
- **`references/http-servers.md`** - HTTP server integration (Fiber, Gin)
- **`references/databases.md`** - Database integration (GORM, sqlc)
- **`references/graceful-shutdown.md`** - Graceful shutdown patterns
- **`references/polyglot-clients.md`** - Python and Node.js client patterns

### Example Files

Working examples in `examples/`:

- **`examples/basic-module.go`** - Minimal module implementation
- **`examples/service-provider.go`** - Service provider with RequestReply
- **`examples/queue-group-service.go`** - QueueGroupService with QGHP pattern
- **`examples/cron-service.go`** - CronService scheduled jobs (interval, cron, alias)
- **`examples/multi-module.go`** - Multi-module with dependencies and service adapters (string-based helpers)
- **`examples/module-boundaries/`** - Recommended layout: contract packages, ports and adapters, `ConsumeServices`, `monotest.CheckModule` test
- **`examples/event-emitter.go`** - Event emitter with consumer
- **`examples/plugin-module.go`** - Custom plugin implementation
- **`examples/kv-jetstream-usecase.go`** - Sessions, counters, distributed locks
- **`examples/fs-jetstream-usecase.go`** - Document storage, uploads, media
- **`examples/middleware-module.go`** - Custom metrics middleware
- **`examples/middleware-usecases.go`** - Using requestid, accesslog, audit
- **`examples/http-fiber-module.go`** - Fiber HTTP server integration
- **`examples/http-gin-module.go`** - Gin HTTP server integration
- **`examples/gorm-sqlite-module.go`** - GORM with SQLite integration
- **`examples/sqlc-postgres-module.go`** - sqlc with PostgreSQL integration

### Example Projects

Real-world implementations in [mono-recipes](https://github.com/go-monolith/mono-recipes/tree/main/projects) collection:

| Project | Demonstrates |
|---------|--------------|
| `hexagonal-architecture` | Ports and Adapters end to end: `contract` packages, consumer-owned ports/adapters, `ConsumeServices`, events in the emitter's contract, `CheckModule` per module |
| `background-jobs-demo` | QueueGroupService with multiple workers via a typed `QueueGroupService` contract |
| `cron-scheduler-demo` | CronService scheduled jobs (interval, cron, alias); stats exposed through a typed contract |
| `file-upload-demo` | fs-jetstream plugin with a Gin HTTP driving adapter inside the module |
| `url-shortener-demo` | kv-jetstream plugin, typed contracts and events, Fiber |
| `jwt-auth-demo` | Authentication with Fiber; API reaches auth through its own port/adapter |
| `websocket-chat-demo` | EventBus patterns with WebSocket; events consumed through contract definitions |
| `python-nats-client-demo` | Python client integration (RequestReply, QueueGroup, StreamConsumer contracts) |
| `node-nats-client-demo` | Node.js client with fs-jetstream |
| `rate-limiting-middleware` | Custom middleware with Redis wrapping typed-contract services |
| `sqlc-postgres-demo` | sqlc with PostgreSQL and health checks |
| `gorm-sqlite-demo` | GORM with SQLite (single connection for concurrent handlers) |
| `redis-caching-plugin` | Custom Redis caching plugin; `monotest.WithPlugin` test |
| `graceful-shutdown-demo` | Draining in-flight HTTP requests and workers within the shutdown timeout |

All recipes target mono **v0.1.1**, run with `mono.WithStrictModuleBoundaries()`, and have a `monotest.CheckModule` (or real-app boundary) test per module.

### Official Documentation

Full documentation available at:
- Getting Started: https://gelmium.gitbook.io/monolith-framework
- Core Concepts: https://gelmium.gitbook.io/monolith-framework#core-concepts
- API Reference: https://gelmium.gitbook.io/monolith-framework/api-reference/api
