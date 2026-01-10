# HTTP Server Integration

Mono applications commonly embed HTTP servers for REST APIs. This reference covers the two most popular frameworks: Fiber and Gin.

## Framework Selection

| Framework | Strengths | Use Cases |
|-----------|-----------|-----------|
| **Fiber** | Fast, Express-like API, built-in middleware | High-performance APIs, familiar syntax for Node.js developers |
| **Gin** | Standard net/http compatible, mature ecosystem | Complex middleware needs, existing net/http compatibility |

## Fiber HTTP Server Module

### Basic Module Structure

```go
package httpserver

import (
    "context"
    "fmt"
    "time"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/types"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/recover"
)

type Module struct {
    app    *fiber.App
    addr   string
    logger types.Logger
}

var _ mono.Module = (*Module)(nil)

func NewModule(addr string, logger types.Logger) *Module {
    return &Module{
        addr:   addr,
        logger: logger,
    }
}

func (m *Module) Name() string { return "http-server" }

func (m *Module) Start(ctx context.Context) error {
    m.app = fiber.New(fiber.Config{
        AppName:               "My API",
        DisableStartupMessage: true,
        ErrorHandler:          m.errorHandler,
    })

    // Middleware stack
    m.app.Use(recover.New())
    m.app.Use(logger.New(logger.Config{
        Format: "[${time}] ${status} ${method} ${path} ${latency}\n",
    }))
    m.app.Use(cors.New())

    // Routes
    m.registerRoutes()

    // Start with startup error detection
    errCh := make(chan error, 1)
    go func() {
        if err := m.app.Listen(m.addr); err != nil {
            errCh <- err
        }
    }()

    select {
    case err := <-errCh:
        return fmt.Errorf("HTTP server failed to start: %w", err)
    case <-time.After(100 * time.Millisecond):
        // Server started successfully
    }

    m.logger.Info("HTTP server started", "addr", m.addr)
    return nil
}

func (m *Module) Stop(ctx context.Context) error {
    if m.app != nil {
        if err := m.app.ShutdownWithContext(ctx); err != nil {
            return fmt.Errorf("failed to shutdown server: %w", err)
        }
    }
    m.logger.Info("HTTP server stopped")
    return nil
}
```

### Route Registration

```go
func (m *Module) registerRoutes() {
    // Health check
    m.app.Get("/health", m.handleHealth)

    // API group with version prefix
    api := m.app.Group("/api/v1")

    // Resource routes
    api.Get("/items", m.handleListItems)
    api.Get("/items/:id", m.handleGetItem)
    api.Post("/items", m.handleCreateItem)
    api.Put("/items/:id", m.handleUpdateItem)
    api.Delete("/items/:id", m.handleDeleteItem)
}

func (m *Module) handleHealth(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{"status": "healthy"})
}
```

### Error Handler

```go
func (m *Module) errorHandler(c *fiber.Ctx, err error) error {
    code := fiber.StatusInternalServerError
    message := "Internal Server Error"

    if e, ok := err.(*fiber.Error); ok {
        code = e.Code
        message = e.Message
    }

    m.logger.Error("HTTP error", "code", code, "error", err)

    return c.Status(code).JSON(fiber.Map{
        "error": message,
    })
}
```

### CORS Configuration

```go
import "os"

// In Start():
allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
if allowedOrigins == "" {
    allowedOrigins = "http://localhost:3000,http://localhost:8080"
}
m.app.Use(cors.New(cors.Config{
    AllowOrigins: allowedOrigins,
    AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
    AllowHeaders: "Content-Type,Authorization",
}))
```

## Gin HTTP Server Module

### Basic Module Structure

```go
package httpserver

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/types"
)

type Module struct {
    port          int
    server        *http.Server
    engine        *gin.Engine
    logger        types.Logger
    maxUploadSize int64
}

var _ mono.Module = (*Module)(nil)

func NewModule(port int, maxUploadSize int64, logger types.Logger) *Module {
    return &Module{
        port:          port,
        maxUploadSize: maxUploadSize,
        logger:        logger,
    }
}

func (m *Module) Name() string { return "http-server" }

func (m *Module) Start(ctx context.Context) error {
    gin.SetMode(gin.ReleaseMode)

    m.engine = gin.New()
    m.engine.Use(gin.Recovery())
    m.engine.Use(m.loggingMiddleware())
    m.engine.Use(m.corsMiddleware())

    // Set max multipart memory for file uploads
    m.engine.MaxMultipartMemory = m.maxUploadSize

    // Register routes
    m.registerRoutes()

    // Create HTTP server with timeouts
    m.server = &http.Server{
        Addr:              fmt.Sprintf(":%d", m.port),
        Handler:           m.engine,
        ReadHeaderTimeout: 10 * time.Second,
        ReadTimeout:       60 * time.Second,
        WriteTimeout:      60 * time.Second,
        IdleTimeout:       120 * time.Second,
    }

    go func() {
        m.logger.Info("HTTP server starting", "port", m.port)
        if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            m.logger.Error("HTTP server error", "error", err)
        }
    }()

    return nil
}

func (m *Module) Stop(ctx context.Context) error {
    if m.server != nil {
        m.logger.Info("Shutting down HTTP server")
        return m.server.Shutdown(ctx)
    }
    return nil
}
```

### Custom Middleware

```go
func (m *Module) loggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        method := c.Request.Method

        c.Next()

        m.logger.Info("HTTP request",
            "method", method,
            "path", path,
            "status", c.Writer.Status(),
            "latency_ms", time.Since(start).Milliseconds(),
            "client_ip", c.ClientIP(),
        )
    }
}

func (m *Module) corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }

        c.Next()
    }
}
```

### Route Registration

```go
func (m *Module) registerRoutes() {
    // Health check
    m.engine.GET("/health", m.handleHealth)

    // API v1 routes
    v1 := m.engine.Group("/api/v1")
    {
        files := v1.Group("/files")
        {
            files.POST("", m.handleUpload)
            files.GET("", m.handleList)
            files.GET("/:id", m.handleGet)
            files.DELETE("/:id", m.handleDelete)
        }
    }
}
```

## Dependency Injection Pattern

HTTP server modules often need access to other module services. Use the DependentModule interface:

```go
type Module struct {
    app              *fiber.App
    addr             string
    shortenerAdapter shortener.ShortenerAdapterPort
    analyticsAdapter analytics.AnalyticsAdapterPort
    logger           types.Logger
}

func (m *Module) Dependencies() []string {
    return []string{"shortener", "analytics"}
}

func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
    switch module {
    case "shortener":
        m.shortenerAdapter = shortener.NewShortenerAdapter(container)
        m.logger.Info("Received shortener service container")
    case "analytics":
        m.analyticsAdapter = analytics.NewAnalyticsAdapter(container)
        m.logger.Info("Received analytics service container")
    }
}
```

## Example Projects

| Project | Framework | Features |
|---------|-----------|----------|
| `url-shortener-demo` | Fiber | CORS, dependency injection, adapter pattern |
| `file-upload-demo` | Gin | Multipart uploads, custom logging middleware |
| `jwt-auth-demo` | Fiber | JWT authentication, protected routes |

## Best Practices

1. **Always use graceful shutdown** - Use `ShutdownWithContext()` to handle in-flight requests
2. **Set appropriate timeouts** - Configure read, write, and idle timeouts
3. **Use release mode in production** - `gin.SetMode(gin.ReleaseMode)`
4. **Disable startup messages** - `DisableStartupMessage: true` for Fiber
5. **Handle startup errors** - Check for port conflicts before returning from Start()
6. **Inject dependencies via interfaces** - Use adapter pattern for loose coupling
7. **Use structured logging** - Pass logger from main.go via constructor
