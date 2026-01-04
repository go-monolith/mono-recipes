---
name: mono-development
description: This skill should be used when the user asks to "create a mono application", "build a mono module", "implement MonoApplication", "add a new module", "setup inter-module communication", "create an event emitter", "register services", "add event consumers", "create a plugin", "use kv-jetstream", "use fs-jetstream", "add file storage", "add key-value storage", "create middleware", "add access logging", "add audit logging", "add request id tracking", or mentions "mono-framework", "MonoApplication", "ServiceContainer", "ChannelService", "RequestReplyService", "QueueGroupService", "StreamConsumerService", "EventBus", "EventRegistry", "EventEmitter", "EventConsumer", "EventStreamConsumer", "PluginModule", "UsePluginModule", "MiddlewareModule", "OnServiceRegistration", "OnModuleLifecycle", "OnOutgoingMessage", "kv-jetstream", "fs-jetstream", "KVStoragePort", "FileStoragePort", "accesslog", "audit", "requestid". Provides best practices for developing modular monolith applications with the go-monolith/mono framework.
version: 0.3.0
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
    "time"
    "github.com/go-monolith/mono"
)

func main() {
    app, err := mono.NewMonoApplication(
        mono.WithLogLevel(mono.LogLevelInfo),
        mono.WithLogFormat(mono.LogFormatText),
        mono.WithShutdownTimeout(10*time.Second),
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

    // Graceful shutdown
    defer app.Stop(ctx)
}
```

### Creating a Basic Module

Every module implements three methods:

```go
type MyModule struct{}

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
| `EventEmitterModule` | Declare events the module emits |
| `EventConsumerModule` | Register event consumer handlers |
| `HealthCheckableModule` | Report custom health status |
| `PluginModule` | Start first, stop last (cross-cutting) |
| `UsePluginModule` | Receive plugin instances |
| `MiddlewareModule` | Intercept and wrap service handlers |

## Service Communication Patterns

Four patterns for inter-module communication:

| Type | Use Case | Latency |
|------|----------|---------|
| **Channel** | In-process, high-throughput | ~microseconds |
| **Request-Reply** | Synchronous with response | ~1ms |
| **Queue Group** | Async, load-balanced | ~1ms |
| **Stream Consumer** | Durable, at-least-once | ~5ms |

### Registering Services (Provider)

```go
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterRequestReplyService(
        "process-payment",
        m.handleProcessPayment,
    )
}

func (m *PaymentModule) handleProcessPayment(
    ctx context.Context, req *mono.Msg) ([]byte, error) {
    // Process and return response
}
```

### Consuming Services (Consumer)

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
    client, _ := m.paymentContainer.GetRequestReplyService("process-payment")
    resp, err := client.Call(ctx, requestData)
    // Handle response
}
```

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

Recommended layout for larger applications:

```
my-app/
├── main.go                  # Application entry point
├── go.mod
├── modules/
│   ├── order/
│   │   ├── module.go       # Module implementation
│   │   ├── adapter.go      # Service adapter (typed client)
│   │   ├── events.go       # Event definitions
│   │   └── types.go        # Domain types
│   ├── payment/
│   │   ├── module.go
│   │   └── types.go
│   └── notification/
│       ├── module.go
│       └── handlers.go
├── config/
│   └── config.go
└── tests/
    └── integration_test.go
```

## Best Practices

### Do

- Keep modules focused on a single domain
- Use services for point-to-point calls (caller knows provider)
- Use events for broadcast/loose coupling (emitter doesn't know consumers)
- Declare dependencies explicitly via `DependentModule`
- Handle errors explicitly, wrap with context using `%w`
- Use structured logging with `log/slog` or pass in `app.Logger()` (builtin framework logger) in module constructor

### Don't

- Call module.Start() or module.Stop() directly (this is handled by framework)
- Define & call modules's function directly - use services or events
- Create circular dependencies - refactor or use events
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

### Using Plugins

```go
// Register plugin with alias
storage, _ := fsjetstream.New(fsjetstream.Config{
    Buckets: []fsjetstream.BucketConfig{
        {Name: "documents", MaxBytes: 1_000_000_000},
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
    m.docs = m.storage.Bucket("documents")
    return nil
}
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
- **`references/events.md`** - Event emitter and consumer patterns
- **`references/plugins.md`** - Plugin creation and built-in plugins
- **`references/middleware.md`** - Middleware hooks and built-in middleware

### Example Files

Working examples in `examples/`:

- **`examples/basic-module.go`** - Minimal module implementation
- **`examples/service-provider.go`** - Service provider with RequestReply
- **`examples/multi-module.go`** - Multi-module with dependencies and service adapters
- **`examples/event-emitter.go`** - Event emitter with consumer
- **`examples/plugin-module.go`** - Custom plugin implementation
- **`examples/kv-jetstream-usecase.go`** - Sessions, counters, distributed locks
- **`examples/fs-jetstream-usecase.go`** - Document storage, uploads, media
- **`examples/middleware-module.go`** - Custom metrics middleware
- **`examples/middleware-usecases.go`** - Using requestid, accesslog, audit

### Official Documentation

Full documentation available at `/workspaces/mono/docs/official/`:

- Getting Started: `getting-started/quickstart.md`
- Core Concepts: `core-concepts/modules.md`, `core-concepts/services.md`
- API Reference: `api/framework.md`, `api/container.md`
