package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/example/file-upload-demo/modules/fileservice"
	"github.com/gin-gonic/gin"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
)

// Module implements an HTTP server using the Gin framework.
type Module struct {
	port          int
	server        *http.Server
	engine        *gin.Engine
	handlers      *Handlers
	fileModule    *fileservice.Module
	logger        types.Logger
	maxUploadSize int64
}

// Compile-time interface checks
var _ mono.Module = (*Module)(nil)

// NewModule creates a new HTTP server module.
func NewModule(port int, maxUploadSize int64, logger types.Logger) *Module {
	return &Module{
		port:          port,
		maxUploadSize: maxUploadSize,
		logger:        logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "http-server"
}

// SetFileModule sets the file service module dependency.
func (m *Module) SetFileModule(fileModule *fileservice.Module) {
	m.fileModule = fileModule
}

// Start initializes and starts the HTTP server.
func (m *Module) Start(ctx context.Context) error {
	if m.fileModule == nil {
		return fmt.Errorf("file-service module not set")
	}

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create Gin engine
	m.engine = gin.New()

	// Add middleware
	m.engine.Use(gin.Recovery())
	m.engine.Use(m.loggingMiddleware())
	m.engine.Use(m.corsMiddleware())

	// Set max multipart memory
	m.engine.MaxMultipartMemory = m.maxUploadSize

	// Create handlers
	m.handlers = NewHandlers(m.fileModule.Service())

	// Register routes
	m.registerRoutes()

	// Create HTTP server
	m.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", m.port),
		Handler:           m.engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		m.logger.Info("HTTP server starting", "port", m.port)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (m *Module) Stop(ctx context.Context) error {
	if m.server != nil {
		m.logger.Info("Shutting down HTTP server")
		return m.server.Shutdown(ctx)
	}
	return nil
}

// registerRoutes sets up all HTTP routes.
func (m *Module) registerRoutes() {
	// Health check
	m.engine.GET("/health", m.handlers.HealthCheck)

	// API v1 routes
	v1 := m.engine.Group("/api/v1")
	{
		files := v1.Group("/files")
		{
			files.POST("", m.handlers.UploadFile)
			files.POST("/batch", m.handlers.UploadMultipleFiles)
			files.GET("", m.handlers.ListFiles)
			files.GET("/:id", m.handlers.GetFile)
			files.GET("/:id/info", m.handlers.GetFileInfo)
			files.DELETE("/:id", m.handlers.DeleteFile)
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
