package fileservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// The Gin HTTP server is the driving adapter of the file-service module. It
// lives inside the module (instead of a separate http-server module) because
// the handlers move binary file contents to and from the storage service,
// which a NATS request-reply contract is not suited for, and a separate module
// would need a direct reference to this one.

// startHTTPServer builds the Gin engine and starts serving on the configured
// port. The listener is opened synchronously so a port already in use fails
// Start instead of only being logged.
func (m *Module) startHTTPServer() error {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create Gin engine
	engine := gin.New()

	// Add middleware
	engine.Use(gin.Recovery())
	engine.Use(m.loggingMiddleware())
	engine.Use(m.corsMiddleware())

	// Set max multipart memory
	engine.MaxMultipartMemory = m.maxUploadSize

	// Register routes
	registerRoutes(engine, NewHandlers(m.service))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", m.port),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("HTTP server failed to listen on %s: %w", server.Addr, err)
	}
	m.server = server

	// Serve in a goroutine
	go func() {
		m.logger.Info("HTTP server starting", "port", m.port)
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.logger.Error("HTTP server error", "error", err)
		}
	}()
	return nil
}

// stopHTTPServer gracefully shuts down the HTTP server.
func (m *Module) stopHTTPServer(ctx context.Context) error {
	if m.server == nil {
		return nil
	}
	m.logger.Info("Shutting down HTTP server")
	return m.server.Shutdown(ctx)
}

// registerRoutes sets up all HTTP routes.
func registerRoutes(engine *gin.Engine, handlers *Handlers) {
	// Health check
	engine.GET("/health", handlers.HealthCheck)

	// API v1 routes
	v1 := engine.Group("/api/v1")
	{
		files := v1.Group("/files")
		{
			files.POST("", handlers.UploadFile)
			files.POST("/batch", handlers.UploadMultipleFiles)
			files.GET("", handlers.ListFiles)
			files.GET("/:id", handlers.GetFile)
			files.GET("/:id/info", handlers.GetFileInfo)
			files.DELETE("/:id", handlers.DeleteFile)
		}
	}
}

// loggingMiddleware provides request logging.
func (m *Module) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		m.logger.Info("HTTP request",
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

// corsMiddleware adds CORS headers for development.
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
