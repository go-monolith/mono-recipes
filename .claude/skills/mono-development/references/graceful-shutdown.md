# Graceful Shutdown Patterns

Mono applications should handle shutdown signals gracefully to complete in-flight requests, close connections, and release resources properly.

## Recommended Package

Use `github.com/gelmium/graceful-shutdown` for production applications:

```go
import gfshutdown "github.com/gelmium/graceful-shutdown"
```

## Pattern 1: Using gelmium/graceful-shutdown (Recommended)

### Full Example

```go
package main

import (
    "context"
    "log"
    "os"
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
    )
    if err != nil {
        log.Fatalf("Failed to create app: %v", err)
    }

    // Register modules
    app.Register(&MyModule{})

    // Start the application
    ctx := context.Background()
    if err := app.Start(ctx); err != nil {
        log.Fatalf("Failed to start: %v", err)
    }

    log.Println("Application started successfully")

    // Graceful shutdown with multiple operations
    wait := gfshutdown.GracefulShutdown(
        context.Background(),
        shutdownTimeout,
        map[string]gfshutdown.Operation{
            "mono-app": func(ctx context.Context) error {
                return app.Stop(ctx)
            },
            // Add other cleanup operations as needed
            "external-connections": func(ctx context.Context) error {
                // Close external connections
                return nil
            },
        },
    )

    exitCode := <-wait
    os.Exit(exitCode)
}
```

### Benefits

- Handles SIGINT and SIGTERM signals automatically
- Executes cleanup operations in parallel
- Respects shutdown timeout
- Returns proper exit codes
- Production-ready error handling

## Pattern 2: Manual Signal Handling

For simpler applications or when you need more control:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/go-monolith/mono"
)

func main() {
    app, err := mono.NewMonoApplication(
        mono.WithShutdownTimeout(10 * time.Second),
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

    log.Println("Application started. Press Ctrl+C to shutdown...")

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    log.Println("Shutting down...")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := app.Stop(shutdownCtx); err != nil {
        log.Fatalf("Failed to stop: %v", err)
    }

    log.Println("Application stopped successfully")
}
```

## Module-Level Shutdown

Modules implement the `Stop(ctx context.Context) error` method for cleanup:

### HTTP Server Shutdown

```go
func (m *APIModule) Stop(ctx context.Context) error {
    if m.app != nil {
        // Fiber
        if err := m.app.ShutdownWithContext(ctx); err != nil {
            return fmt.Errorf("failed to shutdown server: %w", err)
        }
    }
    m.logger.Info("HTTP server stopped")
    return nil
}

// For Gin/net/http
func (m *HTTPModule) Stop(ctx context.Context) error {
    if m.server != nil {
        return m.server.Shutdown(ctx)
    }
    return nil
}
```

### Database Connection Shutdown

```go
// GORM
func (m *DataModule) Stop(_ context.Context) error {
    if m.db != nil {
        sqlDB, err := m.db.DB()
        if err != nil {
            return fmt.Errorf("failed to get sql.DB: %w", err)
        }
        return sqlDB.Close()
    }
    return nil
}

// pgx Pool
func (m *UserModule) Stop(_ context.Context) error {
    if m.pool != nil {
        m.pool.Close()
    }
    return nil
}
```

### Background Worker Shutdown

```go
type WorkerModule struct {
    stopCh chan struct{}
    wg     sync.WaitGroup
}

func (m *WorkerModule) Start(ctx context.Context) error {
    m.stopCh = make(chan struct{})
    m.wg.Add(1)
    go m.runWorker()
    return nil
}

func (m *WorkerModule) runWorker() {
    defer m.wg.Done()
    for {
        select {
        case <-m.stopCh:
            return
        default:
            // Do work
        }
    }
}

func (m *WorkerModule) Stop(ctx context.Context) error {
    close(m.stopCh)

    // Wait for worker to finish with timeout
    done := make(chan struct{})
    go func() {
        m.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return fmt.Errorf("shutdown timed out")
    }
}
```

## Shutdown Order

The Mono framework handles shutdown order automatically:

1. **MiddlewareModule** instances stop last (in reverse registration order)
2. **Regular Module** instances stop in reverse dependency order
3. **PluginModule** instances stop first (in reverse registration order)

This ensures:
- HTTP servers stop accepting new requests first
- Business modules finish processing
- Database connections close
- Plugins (storage, caching) close last

## Configuration

### Framework-Level Timeout

```go
app, _ := mono.NewMonoApplication(
    mono.WithShutdownTimeout(30 * time.Second),
)
```

### Context-Level Timeout

```go
shutdownCtx, cancel := context.WithTimeout(
    context.Background(),
    30 * time.Second,
)
defer cancel()

if err := app.Stop(shutdownCtx); err != nil {
    // Handle timeout or error
}
```

## Best Practices

1. **Set appropriate timeouts** - Match to longest expected operation (e.g., in-flight HTTP requests)
2. **Use context for cancellation** - Pass context through to child operations
3. **Log shutdown progress** - Help diagnose stuck shutdowns
4. **Handle errors gracefully** - Log but don't panic on shutdown errors
5. **Close resources in reverse order** - Stop consumers before producers
6. **Wait for goroutines** - Use sync.WaitGroup to track background work
7. **Test shutdown behavior** - Verify no resource leaks or hanging processes

## Common Timeout Values

| Scenario | Recommended Timeout |
|----------|---------------------|
| Quick APIs | 5-10 seconds |
| Web applications | 15-30 seconds |
| Background jobs | 30-60 seconds |
| Long-running processes | 60+ seconds |

## Example Projects

| Project | Pattern |
|---------|---------|
| `url-shortener-demo` | Manual signal handling |
| `node-nats-client-demo` | gelmium/graceful-shutdown |
| `background-jobs-demo` | Worker with WaitGroup |
