# Service Communication Patterns

Detailed reference for inter-module service communication in the Mono Framework.

## What is a Service?

A service is a named endpoint registered by a module that other modules can invoke. Services are the **public APIs** of modules and establish explicit dependencies between them.

## ServiceContainer Interface

Each module receives its own `ServiceContainer`:

```go
type ServiceContainer interface {
    // Register services (provider side)
    RegisterChannelService(name string, in chan *Msg, out chan *Msg) error
    RegisterRequestReplyService(name string, handler RequestReplyHandler) error
    RegisterQueueGroupService(name string, pairs ...QGHP) error
    RegisterStreamConsumerService(name string, config StreamConsumerConfig, handler StreamConsumerHandler) error

    // Get services (consumer side)
    GetChannelService(serviceName string, consumerModule string) (in chan *Msg, out chan *Msg, err error)
    GetRequestReplyService(name string) (RequestReplyServiceClient, error)
    GetQueueGroupService(name string) (QueueGroupServiceClient, error)
    GetStreamConsumerService(name string) (StreamConsumerServiceClient, error)

    // Query
    Has(name string) bool
    Entries() []*ServiceEntry
}
```

## Service Types

### 1. Channel Services

In-process bidirectional communication using Go channels.

**Characteristics:**
- Lowest latency (~microseconds)
- Single process only
- Bidirectional communication

**Registration:**

```go
func (m *AnalyticsModule) RegisterServices(container mono.ServiceContainer) error {
    in := make(chan *mono.Msg, 100)
    out := make(chan *mono.Msg, 100)

    // Start handler goroutine
    go m.handleAnalytics(in, out)

    return container.RegisterChannelService("analytics-stream", in, out)
}

func (m *AnalyticsModule) handleAnalytics(in, out chan *mono.Msg) {
    for msg := range in {
        // Process message
        result := m.process(msg)
        out <- result
    }
}
```

**Consumption:**

```go
func (m *OrderModule) SetDependencyServiceContainer(
    module string, container mono.ServiceContainer) {
    if module == "analytics" {
        in, out, _ := container.GetChannelService("analytics-stream", m.Name())
        m.analyticsIn = in
        m.analyticsOut = out
    }
}

func (m *OrderModule) sendAnalytics(data []byte) {
    m.analyticsIn <- &mono.Msg{Data: data}
    response := <-m.analyticsOut
    // Handle response
}
```

### 2. Request-Reply Services

Synchronous calls with response via NATS request/reply.

**Characteristics:**
- Supports distribution (~1ms overhead)
- Synchronous with response
- Subject pattern: `services.<module>.<service>`

**Handler Signature:**

```go
type RequestReplyHandler func(ctx context.Context, msg *Msg) ([]byte, error)
```

**Registration:**

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
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }

    result := m.processPayment(ctx, req)

    return json.Marshal(result)
}
```

**Consumption:**

```go
func (m *OrderModule) processOrder(ctx context.Context, order *Order) error {
    client, err := m.paymentContainer.GetRequestReplyService("process-payment")
    if err != nil {
        return fmt.Errorf("payment service not available: %w", err)
    }

    reqData, _ := json.Marshal(PaymentRequest{Amount: order.Total})

    resp, err := client.Call(ctx, reqData)
    if err != nil {
        return fmt.Errorf("payment failed: %w", err)
    }

    var paymentResp PaymentResponse
    json.Unmarshal(resp.Data, &paymentResp)

    return nil
}
```

### 3. Queue Group Services

Asynchronous, load-balanced processing via NATS queue subscriptions.

**Characteristics:**
- Fire-and-forget semantics
- Horizontal scaling across workers
- No response expected

**Handler Signature:**

```go
type QueueGroupHandler func(ctx context.Context, msg *Msg) error
```

**Registration:**

```go
func (m *NotificationModule) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterQueueGroupService(
        "send-notification",
        mono.QGHP{
            QueueGroup: "notification-workers",
            Handler:    m.handleSendNotification,
        },
    )
}

func (m *NotificationModule) handleSendNotification(
    ctx context.Context, msg *mono.Msg) error {
    var req NotificationRequest
    json.Unmarshal(msg.Data, &req)

    // Send email/SMS (fire-and-forget)
    return m.sendEmail(ctx, req.Email, req.Message)
}
```

**Consumption:**

```go
func (m *OrderModule) notifyCustomer(ctx context.Context, email, message string) error {
    client, _ := m.notificationContainer.GetQueueGroupService("send-notification")

    data, _ := json.Marshal(NotificationRequest{
        Email:   email,
        Message: message,
    })

    // Fire-and-forget - no response expected
    return client.Send(ctx, data)
}
```

### 4. Stream Consumer Services

Durable, at-least-once delivery via JetStream pull consumers.

**Characteristics:**
- Messages persisted to JetStream
- Batch processing support
- Explicit acknowledgment (ack/nack)
- Message replay capability

**Handler Signature:**

```go
type StreamConsumerHandler func(ctx context.Context, msgs []*Msg) error
```

**Registration:**

```go
func (m *AuditModule) RegisterServices(container mono.ServiceContainer) error {
    config := mono.StreamConsumerConfig{
        Stream: mono.StreamConfig{
            Name:      "audit-events",
            Retention: mono.WorkQueuePolicy,
        },
        Fetch: mono.FetchConfig{
            BatchSize:   10,
            IdleTimeout: 5 * time.Second,
        },
    }

    return container.RegisterStreamConsumerService(
        "audit-log",
        config,
        m.handleAuditEvents,
    )
}

func (m *AuditModule) handleAuditEvents(
    ctx context.Context, msgs []*mono.Msg) error {
    for _, msg := range msgs {
        var event AuditEvent
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            msg.Nak()  // Retry later
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

**Acknowledgment Methods:**

| Method | Effect |
|--------|--------|
| `msg.Ack()` | Message processed, remove from queue |
| `msg.Nak()` | Retry immediately |
| `msg.NakWithDelay(d)` | Retry after delay |
| `msg.Term()` | Stop redelivery (poison message) |
| `msg.InProgress()` | Extend processing time |

## Dependency Resolution

The framework automatically handles startup order:

```
Modules Registered:
  - OrderModule      → Dependencies: ["payment", "inventory"]
  - PaymentModule    → Dependencies: []
  - InventoryModule  → Dependencies: []
  - ShippingModule   → Dependencies: ["order"]

Computed Startup Order:
  1. PaymentModule     (no dependencies)
  2. InventoryModule   (no dependencies)
  3. OrderModule       (after payment, inventory)
  4. ShippingModule    (after order)

Shutdown Order:
  4 → 3 → 2 → 1       (reverse of startup)
```

### Circular Dependency Detection

Circular dependencies are rejected at startup:

```go
// This will fail:
// OrderModule depends on PaymentModule
// PaymentModule depends on OrderModule

// Error: circular dependency detected: order -> payment -> order
```

## Type-Safe Service Helpers

Use helper functions for type-safe service calls:

```go
import "github.com/go-monolith/mono/pkg/helper"

func (m *OrderModule) processPayment(ctx context.Context, amount float64) error {
    var response PaymentResponse

    err := helper.CallRequestReplyService(
        ctx,
        m.paymentContainer,
        "process-payment",
        json.Marshal,
        json.Unmarshal,
        &PaymentRequest{Amount: amount},
        &response,
    )
    if err != nil {
        return fmt.Errorf("payment failed: %w", err)
    }

    if response.Status != "approved" {
        return fmt.Errorf("payment declined: %s", response.Reason)
    }

    return nil
}
```

## Service Adapter Pattern

Create typed adapters for cleaner inter-module communication:

**Adapter Interface (`adapter.go`):**

```go
package order

type OrderAdapterPort interface {
    PlaceOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)
    GetOrder(ctx context.Context, orderID string) (*Order, error)
}

type orderAdapter struct {
    container mono.ServiceContainer
}

func NewOrderAdapter(container mono.ServiceContainer) OrderAdapterPort {
    return &orderAdapter{container: container}
}

func (a *orderAdapter) PlaceOrder(
    ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
    var response CreateOrderResponse

    err := helper.CallRequestReplyService(
        ctx,
        a.container,
        "create-order",
        json.Marshal,
        json.Unmarshal,
        req,
        &response,
    )

    return &response, err
}
```

**Usage in other modules:**

```go
func (m *ShippingModule) SetDependencyServiceContainer(
    module string, container mono.ServiceContainer) {
    if module == "order" {
        m.orderAdapter = order.NewOrderAdapter(container)
    }
}

func (m *ShippingModule) shipOrder(ctx context.Context, orderID string) error {
    order, err := m.orderAdapter.GetOrder(ctx, orderID)
    if err != nil {
        return err
    }
    // Ship the order
}
```

## Subject Naming Convention

Services are automatically assigned NATS subjects:

```
services.<module>.<service>
```

Examples:
- `services.payment.process-payment`
- `services.inventory.check-stock`
- `services.notification.send-email`

## Services vs Events

| Aspect | Services | Events |
|--------|----------|--------|
| **Coupling** | Tight (declared dependency) | Loose (no dependency) |
| **Direction** | Point-to-point | Broadcast |
| **Response** | Can have response | No response |
| **Discovery** | Via ServiceContainer | Via EventRegistry |
| **Startup Order** | Enforced by framework | Independent |

**Use services when:**
- Response needed from called module
- Known provider module
- Clear caller-callee relationship

**Use events when:**
- Multiple modules might be interested
- Emitter doesn't care who consumes
- Loose coupling desired

## Best Practices

### Do

- Keep services focused (one thing well)
- Use meaningful names (`process-payment` not `handle`)
- Declare all dependencies explicitly
- Handle errors with context (`%w` verb)
- Use appropriate service type for use case

### Don't

- Call modules directly (use services)
- Create circular dependencies
- Share mutable state (use messages)
- Ignore the ServiceContainer
