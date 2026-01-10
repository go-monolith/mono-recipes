# Middleware Module Patterns

Detailed reference for implementing and using middleware in the Mono Framework.

## What is Middleware?

Middleware modules intercept framework events and can observe or modify them before processing. Unlike regular modules, middleware:

- **Start before regular modules**: Ensures middleware is ready to intercept events
- **Stop after regular modules**: Captures all module stop events
- **Chain in registration order**: Multiple middleware execute sequentially
- **Follow decorator pattern**: Can wrap handlers to add behavior

## Middleware Lifecycle

```
1. Framework Start
   │
   ├─→ Plugins start
   │
   ├─→ Middleware start (registration order)
   │
   ├─→ Regular modules start
   │   ├─ OnModuleLifecycle(ModuleStartedEvent)
   │   └─ OnServiceRegistration for each service
   │
   └─→ Application running

2. Framework Stop
   │
   ├─→ Regular modules stop
   │   └─ OnModuleLifecycle(ModuleStoppedEvent)
   │
   └─→ Middleware stop (reverse registration order)
```

## MiddlewareModule Interface

Every middleware implements:

```go
type MiddlewareModule interface {
    Module  // Name(), Start(), Stop()

    // Module lifecycle events (observe or modify)
    OnModuleLifecycle(ctx context.Context, event ModuleLifecycleEvent) ModuleLifecycleEvent

    // Service registration (wrap handlers, modify config)
    OnServiceRegistration(ctx context.Context, reg ServiceRegistration) ServiceRegistration

    // Configuration changes (observe or modify)
    OnConfigurationChange(ctx context.Context, event ConfigurationEvent) ConfigurationEvent

    // Outgoing messages (inject headers, modify payload)
    OnOutgoingMessage(octx OutgoingMessageContext) OutgoingMessageContext

    // Event consumer registration (wrap handlers)
    OnEventConsumerRegistration(ctx context.Context, entry EventConsumerEntry) EventConsumerEntry

    // Event stream consumer registration (wrap handlers)
    OnEventStreamConsumerRegistration(ctx context.Context, entry EventStreamConsumerEntry) EventStreamConsumerEntry
}
```

## Hook Methods

### OnModuleLifecycle

Intercept module start/stop events:

```go
func (m *MyMiddleware) OnModuleLifecycle(
    ctx context.Context,
    event types.ModuleLifecycleEvent,
) types.ModuleLifecycleEvent {
    switch event.Type {
    case types.ModuleStartedEvent:
        // Log startup timing
        slog.Info("Module started",
            "module", event.ModuleName,
            "duration_ms", event.Duration.Milliseconds())
    case types.ModuleStoppedEvent:
        if event.Error != nil {
            slog.Error("Module stop failed",
                "module", event.ModuleName,
                "error", event.Error)
        }
    }
    return event // Pass through (or modify)
}
```

### OnServiceRegistration

Wrap handlers or modify configuration:

```go
func (m *MyMiddleware) OnServiceRegistration(
    ctx context.Context,
    reg types.ServiceRegistration,
) types.ServiceRegistration {
    switch reg.Type {
    case types.ServiceTypeRequestReply:
        if reg.RequestHandler != nil {
            original := reg.RequestHandler
            reg.RequestHandler = func(ctx context.Context, req *types.Msg) ([]byte, error) {
                start := time.Now()
                resp, err := original(ctx, req)
                slog.Debug("Request handled",
                    "service", reg.Name,
                    "duration", time.Since(start))
                return resp, err
            }
        }
    case types.ServiceTypeQueueGroup:
        // Wrap each handler in QueueHandlers
    case types.ServiceTypeStreamConsumer:
        // Wrap StreamHandler and/or modify StreamConsumerConfig
    case types.ServiceTypeChannel:
        // Wrap by replacing InChannel/OutChannel with proxies
    }
    return reg
}
```

### OnOutgoingMessage

Inject headers into outgoing messages:

```go
func (m *MyMiddleware) OnOutgoingMessage(
    octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
    // Ensure header map exists
    if octx.Msg.Header == nil {
        octx.Msg.Header = make(types.Header)
    }

    // Inject trace ID from context
    if traceID := GetTraceID(octx.Ctx); traceID != "" {
        octx.Msg.Header["X-Trace-ID"] = []string{traceID}
    }

    return octx
}
```

### OnEventConsumerRegistration

Wrap event consumer handlers:

```go
func (m *MyMiddleware) OnEventConsumerRegistration(
    ctx context.Context,
    entry types.EventConsumerEntry,
) types.EventConsumerEntry {
    if entry.Handler != nil {
        original := entry.Handler
        entry.Handler = func(ctx context.Context, msg *types.Msg) error {
            start := time.Now()
            err := original(ctx, msg)
            slog.Debug("Event handled",
                "event", entry.EventDef.Name,
                "duration", time.Since(start))
            return err
        }
    }
    return entry
}
```

## Built-in Middleware

### requestid (Request ID Tracking)

Extracts or generates request IDs for tracing across services.

**Features:**
- Extracts X-Request-ID from incoming message headers
- Generates UUID if no request ID present
- Injects request ID into handler context
- Propagates to outgoing messages via OnOutgoingMessage

**Usage:**

```go
import "github.com/go-monolith/mono/middleware/requestid"

middleware, _ := requestid.New(
    requestid.WithHeaderName("X-Request-ID"),  // Default
)
app.Register(middleware)

// Access in handlers
func (m *MyModule) handleRequest(ctx context.Context, req *types.Msg) ([]byte, error) {
    reqID := requestid.GetRequestID(ctx)
    slog.Info("Processing", "request_id", reqID)
    // ...
}
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `WithHeaderName` | "X-Request-ID" | Header key for request ID |

### accesslog (Access Logging)

Logs request/response timing, sizes, and status.

**Features:**
- Wraps all service handlers (RequestReply, QueueGroup, StreamConsumer)
- Captures request timing (before/after handler call)
- Logs request/response sizes and status
- Supports text or JSON output format
- Configurable field selection

**Usage:**

```go
import "github.com/go-monolith/mono/middleware/accesslog"

logFile, _ := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
accessModule, _ := accesslog.New(
    accesslog.WithOutput(logFile),
    accesslog.WithFormat(accesslog.FormatJSON),
    accesslog.WithFields([]accesslog.Field{
        accesslog.FieldTimestamp,
        accesslog.FieldService,
        accesslog.FieldDurationMS,
        accesslog.FieldStatus,
    }),
)
app.Register(accessModule)
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `WithOutput` | required | Output writer (e.g., *os.File) |
| `WithFormat` | FormatText | FormatText or FormatJSON |
| `WithFields` | AllFields() | Fields to include in output |
| `WithRequestIDHeader` | "X-Request-ID" | Header for request ID extraction |

**Available Fields:**
- `FieldTimestamp` - Request timestamp (UTC)
- `FieldRequestID` - Request ID
- `FieldModule` - Module name
- `FieldService` - Service name
- `FieldServiceType` - Service type (request_reply, queue_group, etc.)
- `FieldDurationMS` - Duration in milliseconds
- `FieldStatus` - success/error
- `FieldRequestSize` - Request payload size
- `FieldResponseSize` - Response payload size

### audit (Tamper-Evident Audit Logging)

Provides tamper-evident audit trail with hash chaining.

**Features:**
- Logs module lifecycle events (started, stopped)
- Logs service registrations
- Logs configuration changes
- SHA-256 hash chaining for tamper detection
- JSON output format
- Channel service for custom audit entries

**Usage:**

```go
import "github.com/go-monolith/mono/middleware/audit"

auditFile, _ := os.OpenFile("audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
auditModule, _ := audit.New(
    audit.WithOutput(auditFile),
    audit.WithHashChaining(""),  // Start new chain
    audit.WithUserContext(func(ctx context.Context) string {
        if user, ok := ctx.Value(userKey).(string); ok {
            return user
        }
        return "system"
    }),
)
app.Register(auditModule)  // Register first for full coverage
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `WithOutput` | required | Output writer (e.g., *os.File) |
| `WithHashChaining` | disabled | Enable hash chaining ("" for new chain) |
| `WithUserContext` | returns "" | Function to extract user from context |

**Custom Audit Entries:**

```go
// Via channel service adapter
adapter := audit.NewAdapter(auditContainer)
adapter.SaveEntry(ctx, audit.Entry{
    EventType:   audit.EventCustomAuditTrail,
    ModuleName:  "orders",
    ServiceName: "order-creation",
    Details:     map[string]any{"order_id": "123", "amount": 99.99},
})
```

## Creating Custom Middleware

### Step 1: Define Structure

```go
package mymiddleware

import (
    "context"
    "github.com/go-monolith/mono/pkg/types"
)

type MyMiddleware struct {
    name   string
    config Config
}

type Config struct {
    // Configuration options
}

func New(config Config) (*MyMiddleware, error) {
    return &MyMiddleware{
        name:   "my-middleware",
        config: config,
    }, nil
}
```

### Step 2: Implement Module Interface

```go
func (m *MyMiddleware) Name() string {
    return m.name
}

func (m *MyMiddleware) Start(ctx context.Context) error {
    // Initialize resources
    return nil
}

func (m *MyMiddleware) Stop(ctx context.Context) error {
    // Cleanup resources
    return nil
}
```

### Step 3: Implement MiddlewareModule Hooks

```go
// Compile-time check
var _ mono.MiddlewareModule = (*MyMiddleware)(nil)

func (m *MyMiddleware) OnModuleLifecycle(
    ctx context.Context,
    event types.ModuleLifecycleEvent,
) types.ModuleLifecycleEvent {
    // Observe or modify event
    return event
}

func (m *MyMiddleware) OnServiceRegistration(
    ctx context.Context,
    reg types.ServiceRegistration,
) types.ServiceRegistration {
    // Wrap handlers or modify config
    return reg
}

func (m *MyMiddleware) OnConfigurationChange(
    ctx context.Context,
    event types.ConfigurationEvent,
) types.ConfigurationEvent {
    // Observe or modify config
    return event
}

func (m *MyMiddleware) OnOutgoingMessage(
    octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
    // Modify outgoing messages
    return octx
}

func (m *MyMiddleware) OnEventConsumerRegistration(
    ctx context.Context,
    entry types.EventConsumerEntry,
) types.EventConsumerEntry {
    // Wrap event consumer handlers
    return entry
}

func (m *MyMiddleware) OnEventStreamConsumerRegistration(
    ctx context.Context,
    entry types.EventStreamConsumerEntry,
) types.EventStreamConsumerEntry {
    // Wrap event stream consumer handlers
    return entry
}
```

## Middleware Patterns

### Observer Pattern (Audit)

Pass events through unchanged, just log them:

```go
func (m *AuditMiddleware) OnServiceRegistration(
    ctx context.Context,
    reg types.ServiceRegistration,
) types.ServiceRegistration {
    m.logServiceRegistered(reg.Name, reg.Type)
    return reg  // Pass through unchanged
}
```

### Decorator Pattern (Access Log)

Wrap handlers to add behavior:

```go
func (m *AccessLogMiddleware) OnServiceRegistration(
    ctx context.Context,
    reg types.ServiceRegistration,
) types.ServiceRegistration {
    if reg.RequestHandler != nil {
        original := reg.RequestHandler
        reg.RequestHandler = func(ctx context.Context, req *types.Msg) ([]byte, error) {
            start := time.Now()
            resp, err := original(ctx, req)
            m.logAccess(reg.Name, time.Since(start), err)
            return resp, err
        }
    }
    return reg
}
```

### Context Injection Pattern (Request ID)

Add values to context and propagate through system:

```go
func (m *RequestIDMiddleware) wrapHandler(
    original types.RequestReplyHandler,
) types.RequestReplyHandler {
    return func(ctx context.Context, req *types.Msg) ([]byte, error) {
        requestID := m.extractOrGenerate(req)
        ctx = context.WithValue(ctx, requestIDKey, requestID)
        return original(ctx, req)
    }
}

func (m *RequestIDMiddleware) OnOutgoingMessage(
    octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
    requestID := GetRequestID(octx.Ctx)
    if requestID != "" {
        if octx.Msg.Header == nil {
            octx.Msg.Header = make(types.Header)
        }
        octx.Msg.Header["X-Request-ID"] = []string{requestID}
    }
    return octx
}
```

## Registration Order

Middleware executes in registration order:

```go
app.Register(auditMiddleware)      // First: observe all events
app.Register(requestIDMiddleware)  // Second: inject request IDs
app.Register(accessLogMiddleware)  // Third: log with request IDs

app.Register(myModule)  // Regular modules after middleware
```

**Recommendation:**
1. Audit middleware first (observe everything)
2. Request ID middleware (inject IDs for logging)
3. Access log middleware (uses request IDs)
4. Custom middleware as needed

## Best Practices

### Do

- Register middleware before regular modules
- Use observer pattern when only logging (no modification)
- Use decorator pattern when wrapping handlers
- Pass events through unchanged unless modification needed
- Handle nil handlers gracefully
- Clean up resources in Stop()

### Don't

- Modify events unless necessary
- Block in hooks (use goroutines for async work)
- Panic in hooks (log errors instead)
- Store references to handlers (wrap immediately)
- Forget to check for nil before wrapping
- Register middleware after regular modules

## Complete Custom Middleware Example

```go
package metrics

import (
    "context"
    "sync/atomic"
    "time"
    "github.com/go-monolith/mono/pkg/types"
)

type MetricsMiddleware struct {
    name         string
    requestCount atomic.Int64
    errorCount   atomic.Int64
    totalLatency atomic.Int64
}

var _ mono.MiddlewareModule = (*MetricsMiddleware)(nil)

func New() *MetricsMiddleware {
    return &MetricsMiddleware{name: "metrics"}
}

func (m *MetricsMiddleware) Name() string { return m.name }

func (m *MetricsMiddleware) Start(_ context.Context) error { return nil }

func (m *MetricsMiddleware) Stop(_ context.Context) error { return nil }

func (m *MetricsMiddleware) OnModuleLifecycle(
    _ context.Context,
    event types.ModuleLifecycleEvent,
) types.ModuleLifecycleEvent {
    return event
}

func (m *MetricsMiddleware) OnServiceRegistration(
    _ context.Context,
    reg types.ServiceRegistration,
) types.ServiceRegistration {
    if reg.Type == types.ServiceTypeRequestReply && reg.RequestHandler != nil {
        original := reg.RequestHandler
        reg.RequestHandler = func(ctx context.Context, req *types.Msg) ([]byte, error) {
            m.requestCount.Add(1)
            start := time.Now()
            resp, err := original(ctx, req)
            m.totalLatency.Add(time.Since(start).Milliseconds())
            if err != nil {
                m.errorCount.Add(1)
            }
            return resp, err
        }
    }
    return reg
}

func (m *MetricsMiddleware) OnConfigurationChange(
    _ context.Context,
    event types.ConfigurationEvent,
) types.ConfigurationEvent {
    return event
}

func (m *MetricsMiddleware) OnOutgoingMessage(
    octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
    return octx
}

func (m *MetricsMiddleware) OnEventConsumerRegistration(
    _ context.Context,
    entry types.EventConsumerEntry,
) types.EventConsumerEntry {
    return entry
}

func (m *MetricsMiddleware) OnEventStreamConsumerRegistration(
    _ context.Context,
    entry types.EventStreamConsumerEntry,
) types.EventStreamConsumerEntry {
    return entry
}

// Metrics getters
func (m *MetricsMiddleware) RequestCount() int64  { return m.requestCount.Load() }
func (m *MetricsMiddleware) ErrorCount() int64    { return m.errorCount.Load() }
func (m *MetricsMiddleware) AverageLatencyMS() float64 {
    count := m.requestCount.Load()
    if count == 0 {
        return 0
    }
    return float64(m.totalLatency.Load()) / float64(count)
}
```
