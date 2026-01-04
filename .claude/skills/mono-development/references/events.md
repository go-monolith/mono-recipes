# Event-Driven Communication

Detailed reference for event emitters and consumers in the Mono Framework.

## Events vs Services

Events enable **loose coupling** - consumers do NOT create dependencies on emitters.

| Aspect | Services | Events |
|--------|----------|--------|
| **Dependency** | Consumer must declare dependency | No dependency required |
| **Coupling** | Tight (explicit) | Loose (no coupling) |
| **Direction** | Point-to-point | Broadcast to all interested |
| **Discovery** | Via ServiceContainer | Via EventRegistry |
| **Startup Order** | Provider before consumer | Independent |
| **Knowledge** | Consumer knows provider | Emitter doesn't know consumers |

## Event Consumer Patterns

Two patterns for consuming events:

| Pattern | Durability | Use Case |
|---------|------------|----------|
| **EventConsumer** | None (fire-and-forget) | Real-time, low latency |
| **EventStreamConsumer** | JetStream (at-least-once) | Critical events, audit |

## Defining Events

Use the helper package for type-safe event definitions:

```go
import "github.com/go-monolith/mono/pkg/helper"

// Define typed event
var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEvent](
    "order",        // Module name (domain)
    "OrderCreated", // Event name
    "v1",           // Version
)

// Event payload struct
type OrderCreatedEvent struct {
    OrderID    string    `json:"order_id"`
    CustomerID string    `json:"customer_id"`
    Amount     float64   `json:"amount"`
    Currency   string    `json:"currency"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### Multiple Event Versions

Support event evolution with versioning:

```go
var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEventV1](
    "order", "OrderCreated", "v1",
)

var OrderCreatedV2 = helper.EventDefinition[OrderCreatedEventV2](
    "order", "OrderCreated", "v2",
)

// V1 payload (legacy)
type OrderCreatedEventV1 struct {
    OrderID string  `json:"order_id"`
    Amount  float64 `json:"amount"`
}

// V2 payload (new fields)
type OrderCreatedEventV2 struct {
    OrderID    string    `json:"order_id"`
    CustomerID string    `json:"customer_id"`
    Amount     float64   `json:"amount"`
    Currency   string    `json:"currency"`
    Items      []Item    `json:"items"`
}
```

## Implementing Event Emitters

### Step 1: Implement EventEmitterModule

```go
type OrderModule struct {
    eventBus mono.EventBus
}

// Implement EventBusAwareModule (embedded in EventEmitterModule)
func (m *OrderModule) SetEventBus(bus mono.EventBus) {
    m.eventBus = bus
}

// Declare events this module emits
func (m *OrderModule) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        OrderCreatedV1.ToBase(),
        OrderShippedV1.ToBase(),
        OrderCancelledV1.ToBase(),
    }
}
```

### Step 2: Publish Events

**Fire-and-forget (NATS Core):**

```go
func (m *OrderModule) createOrder(ctx context.Context, order *Order) error {
    // Business logic...

    // Publish event (fire-and-forget)
    err := OrderCreatedV1.Publish(m.eventBus, OrderCreatedEvent{
        OrderID:    order.ID,
        CustomerID: order.CustomerID,
        Amount:     order.Total,
        Currency:   order.Currency,
        CreatedAt:  time.Now(),
    }, nil)

    if err != nil {
        slog.Error("Failed to publish event", "error", err)
        // Continue - event publishing failure shouldn't fail order creation
    }

    return nil
}
```

**Durable (JetStream):**

```go
func (m *OrderModule) createOrder(ctx context.Context, order *Order) error {
    // Business logic...

    // Publish to JetStream (persisted)
    ack, err := OrderCreatedV1.EventStreamPublish(ctx, m.eventBus, OrderCreatedEvent{
        OrderID:    order.ID,
        CustomerID: order.CustomerID,
        Amount:     order.Total,
    }, nil)

    if err != nil {
        return fmt.Errorf("failed to publish event: %w", err)
    }

    slog.Info("Event published",
        "sequence", ack.Sequence(),
        "stream", ack.Stream())

    return nil
}
```

## Implementing Event Consumers

### Pattern 1: EventConsumer (NATS Core)

Fire-and-forget consumption with ~1ms latency.

**When to use:**
- Real-time notifications
- UI updates
- Non-critical events where occasional loss is acceptable

**Raw Handler:**

```go
func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    // Discover event (no dependency on emitter)
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
        return fmt.Errorf("unmarshal failed: %w", err)
    }

    return m.sendOrderConfirmation(ctx, event.CustomerID, event.OrderID)
}
```

**Type-Safe Handler:**

```go
import "github.com/go-monolith/mono/pkg/helper"

func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    // Type-safe registration with automatic unmarshaling
    return helper.RegisterTypedEventConsumer(
        registry,
        order.OrderCreatedV1,  // Import event definition
        m.handleOrderCreated,
        m,
    )
}

// Handler receives pre-deserialized event
func (m *NotificationModule) handleOrderCreated(
    ctx context.Context,
    event order.OrderCreatedEvent,  // Already unmarshaled!
    msg *mono.Msg,
) error {
    return m.sendOrderConfirmation(ctx, event.CustomerID, event.OrderID)
}
```

**With Queue Group (load balancing):**

```go
return registry.RegisterEventConsumer(
    eventDef,
    m.handleOrderCreated,
    m,
    "notification-workers",  // Queue group name
)
```

### Pattern 2: EventStreamConsumer (JetStream)

Durable consumption with at-least-once delivery.

**When to use:**
- Critical business events (payments, orders)
- Audit trails and compliance
- Events where loss is unacceptable
- Batch processing

**Raw Handler:**

```go
func (m *AuditModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    eventDef, ok := registry.GetEventByName("OrderCreated", "v1", "order")
    if !ok {
        return fmt.Errorf("event not found")
    }

    config := mono.StreamConsumerConfig{
        Stream: mono.StreamConfig{
            Name:      "audit-order-events",
            Retention: mono.WorkQueuePolicy,
        },
        Fetch: mono.FetchConfig{
            BatchSize:   10,
            IdleTimeout: 5 * time.Second,
        },
    }

    return registry.RegisterEventStreamConsumer(
        eventDef,
        config,
        m.handleOrderEvents,
        m,
    )
}

func (m *AuditModule) handleOrderEvents(
    ctx context.Context, msgs []*mono.Msg) error {
    for _, msg := range msgs {
        var event OrderCreatedEvent
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            msg.Nak()  // Retry immediately
            continue
        }

        if err := m.writeAuditLog(ctx, event); err != nil {
            msg.NakWithDelay(5 * time.Second)  // Retry with delay
            continue
        }

        msg.Ack()  // Successfully processed
    }
    return nil
}
```

**Type-Safe Handler:**

```go
import "github.com/go-monolith/mono/pkg/helper"

func (m *AuditModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    config := mono.StreamConsumerConfig{
        Stream: mono.StreamConfig{Name: "audit-events"},
        Fetch:  mono.FetchConfig{BatchSize: 10},
    }

    return helper.RegisterTypedEventStreamConsumer(
        registry,
        order.OrderCreatedV1,
        config,
        m.handleOrderEvents,
        m,
    )
}

// Handler receives pre-deserialized events
func (m *AuditModule) handleOrderEvents(
    ctx context.Context,
    events []order.OrderCreatedEvent,  // Already unmarshaled!
    msgs []*mono.Msg,
) error {
    for i, event := range events {
        if err := m.writeAuditLog(ctx, event); err != nil {
            msgs[i].NakWithDelay(5 * time.Second)
            continue
        }
        msgs[i].Ack()
    }
    return nil
}
```

## Message Acknowledgment

JetStream consumers must explicitly acknowledge:

| Method | Effect |
|--------|--------|
| `msg.Ack()` | Processed successfully, remove from queue |
| `msg.Nak()` | Failed, retry immediately |
| `msg.NakWithDelay(d)` | Failed, retry after delay |
| `msg.Term()` | Poison message, stop redelivery |
| `msg.InProgress()` | Extend processing time, prevent timeout |

### Error Handling Patterns

```go
func (m *AuditModule) handleOrderEvents(
    ctx context.Context, msgs []*mono.Msg) error {
    for _, msg := range msgs {
        var event OrderCreatedEvent
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            // Invalid message - terminate to avoid infinite retries
            slog.Error("Invalid event data", "error", err)
            msg.Term()
            continue
        }

        err := m.processEvent(ctx, event)
        switch {
        case err == nil:
            msg.Ack()

        case errors.Is(err, ErrTemporaryFailure):
            // Temporary failure - retry with backoff
            msg.NakWithDelay(5 * time.Second)

        case errors.Is(err, ErrPermanentFailure):
            // Permanent failure - don't retry
            slog.Error("Permanent failure", "error", err, "orderID", event.OrderID)
            msg.Term()

        default:
            // Unknown error - retry
            msg.Nak()
        }
    }
    return nil
}
```

## EventRegistry Interface

The `EventRegistry` provides event discovery:

```go
type EventRegistry interface {
    // Event Discovery
    GetEventByName(name, version, moduleName string) (BaseEventDefinition, bool)
    GetEventsByModule(moduleName string) []BaseEventDefinition
    GetAllEvents() []BaseEventDefinition

    // Consumer Registration
    RegisterEventConsumer(eventDef, handler, module, queueGroup...) error
    RegisterEventStreamConsumer(eventDef, config, handler, module) error
}
```

### Event Discovery Methods

```go
// Find specific event
event, found := registry.GetEventByName("OrderCreated", "v1", "order")

// List all events from a module
events := registry.GetEventsByModule("order")

// List all registered events
allEvents := registry.GetAllEvents()
```

## Event Discovery Patterns

### Pattern A: Direct Import (Type-Safe)

```go
import "myapp/modules/order"  // Import emitter module

func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    return helper.RegisterTypedEventConsumer(
        registry,
        order.OrderCreatedV1,  // Direct reference
        m.handleOrderCreated,
        m,
    )
}
```

**Pros:** Compile-time type safety, IDE support
**Cons:** Creates import dependency (not runtime dependency)

### Pattern B: Runtime Discovery (Decoupled)

```go
func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    eventDef, ok := registry.GetEventByName("OrderCreated", "v1", "order")
    if !ok {
        return fmt.Errorf("event not found")
    }

    return registry.RegisterEventConsumer(eventDef, m.handleOrderCreated, m)
}
```

**Pros:** No import dependency, fully decoupled
**Cons:** No compile-time type checking, runtime errors possible

## Complete Example

### Event Emitter Module

```go
package tracking

import (
    "context"
    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/helper"
)

// Event definitions
var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEvent](
    "tracking", "OrderCreated", "v1",
)

var OrderShippedV1 = helper.EventDefinition[OrderShippedEvent](
    "tracking", "OrderShipped", "v1",
)

type OrderCreatedEvent struct {
    OrderID    string  `json:"order_id"`
    CustomerID string  `json:"customer_id"`
    Amount     float64 `json:"amount"`
}

type OrderShippedEvent struct {
    OrderID     string `json:"order_id"`
    TrackingNum string `json:"tracking_num"`
    Carrier     string `json:"carrier"`
}

type TrackingModule struct {
    eventBus mono.EventBus
}

var _ mono.EventEmitterModule = (*TrackingModule)(nil)

func (m *TrackingModule) Name() string { return "tracking" }
func (m *TrackingModule) Start(_ context.Context) error { return nil }
func (m *TrackingModule) Stop(_ context.Context) error { return nil }

func (m *TrackingModule) SetEventBus(bus mono.EventBus) {
    m.eventBus = bus
}

func (m *TrackingModule) EmitEvents() []mono.BaseEventDefinition {
    return []mono.BaseEventDefinition{
        OrderCreatedV1.ToBase(),
        OrderShippedV1.ToBase(),
    }
}

func (m *TrackingModule) CreateOrder(customerID string, amount float64) (string, error) {
    orderID := generateOrderID()

    // Publish event
    OrderCreatedV1.Publish(m.eventBus, OrderCreatedEvent{
        OrderID:    orderID,
        CustomerID: customerID,
        Amount:     amount,
    }, nil)

    return orderID, nil
}
```

### Event Consumer Module

```go
package notification

import (
    "context"
    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/helper"
    "myapp/modules/tracking"
)

type NotificationModule struct{}

var _ mono.EventConsumerModule = (*NotificationModule)(nil)

func (m *NotificationModule) Name() string { return "notification" }
func (m *NotificationModule) Start(_ context.Context) error { return nil }
func (m *NotificationModule) Stop(_ context.Context) error { return nil }

func (m *NotificationModule) RegisterEventConsumers(
    registry mono.EventRegistry) error {
    // Type-safe registration
    return helper.RegisterTypedEventConsumer(
        registry,
        tracking.OrderCreatedV1,
        m.handleOrderCreated,
        m,
    )
}

func (m *NotificationModule) handleOrderCreated(
    ctx context.Context,
    event tracking.OrderCreatedEvent,
    msg *mono.Msg,
) error {
    slog.Info("Sending order confirmation",
        "orderID", event.OrderID,
        "customerID", event.CustomerID)
    return nil
}
```

## Best Practices

### Do

- Use `EventStreamConsumer` for critical events
- Keep handlers idempotent (events may be redelivered)
- Use queue groups for scaling consumers
- Version events for evolution (`v1`, `v2`)
- Handle errors gracefully (log and ack/nack appropriately)

### Don't

- Create circular imports (use runtime discovery if needed)
- Rely on event ordering (may arrive out of order)
- Store EventRegistry as field (use only in `RegisterEventConsumers`)
- Forget to acknowledge in EventStreamConsumer
- Assume single delivery (design for at-least-once)
