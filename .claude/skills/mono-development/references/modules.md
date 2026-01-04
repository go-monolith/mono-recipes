# Module Interface Patterns

Detailed reference for implementing modules in the Mono Framework.

## Core Module Interface

Every module must implement the base `Module` interface:

```go
type Module interface {
    Name() string                    // Unique identifier
    Start(context.Context) error     // Called on startup
    Stop(context.Context) error      // Called on shutdown
}
```

### Minimal Implementation

```go
type MyModule struct{}

func (m *MyModule) Name() string               { return "my-module" }
func (m *MyModule) Start(_ context.Context) error { return nil }
func (m *MyModule) Stop(_ context.Context) error  { return nil }
```

## Module Lifecycle

The framework manages a strict lifecycle for each module:

```
1. Application Start
   │
   ├─→ Resolve dependencies (topological sort)
   │
   ├─→ For each module in dependency order:
   │   ├─ Create ServiceContainer
   │   ├─ Call SetDependencyServiceContainer()
   │   ├─ Call SetEventBus()
   │   ├─ Call RegisterServices()
   │   ├─ Call Start()
   │   └─ Set up NATS subscriptions
   │
   └─ Application Ready

2. Running
   │
   └─ Modules process events and requests

3. Application Stop
   │
   ├─→ For each module in REVERSE dependency order:
   │   ├─ Drain subscriptions
   │   └─ Call Stop()
   │
   └─ Shutdown Complete
```

## Optional Interfaces

### EventBusAwareModule

Receive the EventBus for publishing messages:

```go
type EventBusAwareModule interface {
    Module
    SetEventBus(EventBus)
}
```

**Implementation:**

```go
type MyModule struct {
    eventBus mono.EventBus
}

func (m *MyModule) SetEventBus(bus mono.EventBus) {
    m.eventBus = bus
}
```

**When to use:** When publishing events or making direct NATS calls.

### ServiceProviderModule

Register services the module provides:

```go
type ServiceProviderModule interface {
    Module
    RegisterServices(ServiceContainer) error
}
```

**Implementation:**

```go
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
    // Register RequestReply service
    if err := container.RegisterRequestReplyService(
        "process-payment",
        m.handleProcessPayment,
    ); err != nil {
        return err
    }

    // Register QueueGroup service
    return container.RegisterQueueGroupService(
        "send-receipt",
        mono.QGHP{
            QueueGroup: "receipt-workers",
            Handler:    m.handleSendReceipt,
        },
    )
}
```

**When to use:** When other modules need to call your services.

### DependentModule

Declare dependencies on other modules:

```go
type DependentModule interface {
    Module
    Dependencies() []string
}
```

**Implementation:**

```go
func (m *OrderModule) Dependencies() []string {
    return []string{"payment", "inventory"}
}
```

**Effect:** Framework ensures `payment` and `inventory` modules start before `order`.

**When to use:** When your module needs other modules' services to function.

### SetDependencyServiceContainer

Access services from dependency modules:

```go
type SetDependencyServiceContainerModule interface {
    Module
    SetDependencyServiceContainer(module string, container ServiceContainer)
}
```

**Implementation:**

```go
type OrderModule struct {
    paymentContainer   mono.ServiceContainer
    inventoryContainer mono.ServiceContainer
}

func (m *OrderModule) SetDependencyServiceContainer(
    module string, container mono.ServiceContainer) {
    switch module {
    case "payment":
        m.paymentContainer = container
    case "inventory":
        m.inventoryContainer = container
    }
}
```

**When to use:** When consuming services provided by dependency modules.

### EventEmitterModule

Declare events the module emits (extends EventBusAwareModule):

```go
type EventEmitterModule interface {
    EventBusAwareModule
    EmitEvents() []BaseEventDefinition
}
```

**Implementation:**

```go
import "github.com/go-monolith/mono/pkg/helper"

var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEvent](
    "order", "OrderCreated", "v1",
)

var OrderShippedV1 = helper.EventDefinition[OrderShippedEvent](
    "order", "OrderShipped", "v1",
)

func (m *OrderModule) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        OrderCreatedV1.ToBase(),
        OrderShippedV1.ToBase(),
    }
}
```

**When to use:** When publishing events for other modules to consume.

### EventConsumerModule

Register event consumer handlers:

```go
type EventConsumerModule interface {
    Module
    RegisterEventConsumers(EventRegistry) error
}
```

**Implementation:**

```go
func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    // Discover event by name (no dependency on emitter module)
    eventDef, ok := registry.GetEventByName("OrderCreated", "v1", "order")
    if !ok {
        return fmt.Errorf("event OrderCreated.v1 not found")
    }

    return registry.RegisterEventConsumer(
        eventDef,
        m.handleOrderCreated,
        m,
    )
}

func (m *NotificationModule) handleOrderCreated(
    ctx context.Context, msg *mono.Msg) error {
    var event OrderCreatedEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        return err
    }
    return m.sendNotification(ctx, event)
}
```

**When to use:** When reacting to events from other modules.

### HealthCheckableModule

Report custom health status:

```go
type HealthCheckableModule interface {
    Module
    Health(context.Context) HealthStatus
}
```

**Implementation:**

```go
func (m *DatabaseModule) Health(ctx context.Context) mono.HealthStatus {
    if err := m.db.PingContext(ctx); err != nil {
        return mono.HealthStatus{
            Healthy: false,
            Status:  "Database connection failed",
        }
    }
    return mono.HealthStatus{
        Healthy: true,
        Status:  "Connected",
    }
}
```

**When to use:** When reporting custom health conditions (database, external APIs).

### PluginModule

Special modules that start first and stop last:

```go
type PluginModule interface {
    Module
}
```

**Registration:**

```go
storagePlugin, _ := fsjetstream.New(config)
app.RegisterPlugin(storagePlugin, "storage")  // alias for lookup
```

**When to use:** For cross-cutting concerns like storage, caching, authentication.

### UsePluginModule

Access plugin instances:

```go
type UsePluginModule interface {
    Module
    SetPlugin(alias string, plugin PluginModule)
}
```

**Implementation:**

```go
type MyModule struct {
    storage *fsjetstream.PluginModule
}

func (m *MyModule) SetPlugin(alias string, plugin mono.PluginModule) {
    if alias == "storage" {
        m.storage = plugin.(*fsjetstream.PluginModule)
    }
}
```

**When to use:** When using functionality from a registered plugin.

## Compile-Time Interface Checks

Always use compile-time interface checks to catch implementation errors early:

```go
// Compile-time interface checks.
var _ mono.Module = (*MyModule)(nil)
// or 
var _ mono.EventBusAwareModule = (*MyModule)(nil)
// or
var _ mono.ServiceProviderModule = (*MyModule)(nil)
```

**Why use this pattern:**

1. **Early error detection**: Catches missing methods at compile time, not runtime
2. **Self-documenting**: Clearly shows which interfaces the type implements
3. **Refactoring safety**: If interface changes, compiler immediately flags issues
4. **No runtime cost**: The `var _ =` pattern is optimized away by the compiler

**Standard format:**

```go
// For plugins
var _ mono.PluginModule = (*MyPlugin)(nil)

// For modules implementing multiple interfaces
// Note that: all module interfaces already inherit from base Module interface
// and some module interfaces inherit from others. For example, EventEmitterModule inherit from EventBusAwareModule.
// When specifying multiple interfaces, only include each unique interface once. No need to repeat base interfaces.
// Or interfaces that are already inherited by other interfaces.
var (
    _ mono.ServiceProviderModule = (*MyModule)(nil)
    _ mono.DependentModule       = (*MyModule)(nil)
    _ mono.UsePluginModule       = (*MyModule)(nil)
    _ mono.EventEmitterModule    = (*MyModule)(nil)
)

// For middleware
var _ mono.MiddlewareModule = (*MyMiddleware)(nil)
```

**Placement**: Put compile-time checks immediately after the struct definition or constructor function, before method implementations.

## Complete Module Example

Module implementing multiple interfaces:

```go
package order

import (
    "context"
    "log/slog"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/helper"
)

// Event definitions
var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEvent](
    "order", "OrderCreated", "v1",
)

type OrderCreatedEvent struct {
    OrderID    string  `json:"order_id"`
    CustomerID string  `json:"customer_id"`
    Amount     float64 `json:"amount"`
}

// Module struct
type Module struct {
    eventBus         mono.EventBus
    paymentContainer mono.ServiceContainer
}

// Compile-time interface checks
var (
    _ mono.Module                              = (*Module)(nil)
    _ mono.EventBusAwareModule                 = (*Module)(nil)
    _ mono.ServiceProviderModule               = (*Module)(nil)
    _ mono.DependentModule                     = (*Module)(nil)
    _ mono.SetDependencyServiceContainerModule = (*Module)(nil)
    _ mono.EventEmitterModule                  = (*Module)(nil)
    _ mono.HealthCheckableModule               = (*Module)(nil)
)

func NewModule() *Module {
    return &Module{}
}

// Required: Module interface
func (m *Module) Name() string { return "order" }

func (m *Module) Start(ctx context.Context) error {
    slog.Info("Starting order module")
    return nil
}

func (m *Module) Stop(ctx context.Context) error {
    slog.Info("Stopping order module")
    return nil
}

// Optional: EventBusAwareModule
func (m *Module) SetEventBus(bus mono.EventBus) {
    m.eventBus = bus
}

// Optional: DependentModule
func (m *Module) Dependencies() []string {
    return []string{"payment"}
}

// Optional: SetDependencyServiceContainer
func (m *Module) SetDependencyServiceContainer(
    module string, container mono.ServiceContainer) {
    if module == "payment" {
        m.paymentContainer = container
    }
}

// Optional: ServiceProviderModule
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterRequestReplyService(
        "create-order",
        m.handleCreateOrder,
    )
}

func (m *Module) handleCreateOrder(
    ctx context.Context, req *mono.Msg) ([]byte, error) {
    // Business logic
    return json.Marshal(response)
}

// Optional: EventEmitterModule
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        OrderCreatedV1.ToBase(),
    }
}

// Optional: HealthCheckableModule
func (m *Module) Health(ctx context.Context) mono.HealthStatus {
    return mono.HealthStatus{Healthy: true, Status: "Ready"}
}
```

## Module Registration

Register modules in the main application:

```go
app, _ := mono.NewMonoApplication()

// Register modules (framework resolves dependency order)
app.Register(&OrderModule{})
app.Register(&PaymentModule{})
app.Register(&NotificationModule{})

// Start (modules initialized in dependency order)
app.Start(context.Background())
```

## Error Handling in Modules

### Start Errors

If `Start()` returns an error, the framework:
1. Stops all previously started modules in reverse order
2. Returns error with context about which module failed

```go
func (m *MyModule) Start(ctx context.Context) error {
    db, err := m.connectDatabase(ctx)
    if err != nil {
        return fmt.Errorf("database connect failed: %w", err)
    }
    m.db = db
    return nil
}
```

### Stop Errors

`Stop()` errors are logged but don't prevent other modules from stopping:

```go
func (m *MyModule) Stop(ctx context.Context) error {
    if m.db != nil {
        if err := m.db.Close(); err != nil {
            return fmt.Errorf("database close failed: %w", err)
        }
    }
    return nil
}
```

## Best Practices

### Do

- Use compile-time interface checks (`var _ mono.Module = (*MyModule)(nil)`)
- Keep modules focused on a single domain
- Use structured logging with `log/slog`
- Handle errors explicitly with context
- Implement `HealthCheckableModule` for external dependencies

### Don't

- Store global state in module packages
- Call other modules directly (use services or events)
- Block in `Start()` on I/O without timeout
- Panic in modules (return errors instead)
- Create circular dependencies
