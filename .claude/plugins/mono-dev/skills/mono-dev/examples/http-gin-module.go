// http-gin-module.go demonstrates HTTP server integration using Gin framework
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

// Module implements HTTP server using Gin framework
type Module struct {
	port          int
	server        *http.Server
	engine        *gin.Engine
	logger        types.Logger
	maxUploadSize int64
}

// Compile-time interface check
var _ mono.Module = (*Module)(nil)

// NewModule creates a new HTTP server module
func NewModule(port int, maxUploadSize int64, logger types.Logger) *Module {
	return &Module{
		port:          port,
		maxUploadSize: maxUploadSize,
		logger:        logger,
	}
}

// Name returns the module name
func (m *Module) Name() string { return "http-server" }

// Start initializes and starts the HTTP server
func (m *Module) Start(ctx context.Context) error {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create Gin engine
	m.engine = gin.New()

	// Add middleware
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

	// Start server in goroutine
	go func() {
		m.logger.Info("HTTP server starting", "port", m.port)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server
func (m *Module) Stop(ctx context.Context) error {
	if m.server != nil {
		m.logger.Info("Shutting down HTTP server")
		return m.server.Shutdown(ctx)
	}
	return nil
}

// registerRoutes sets up all HTTP routes
func (m *Module) registerRoutes() {
	// Health check
	m.engine.GET("/health", m.handleHealth)

	// API v1 routes
	v1 := m.engine.Group("/api/v1")
	{
		files := v1.Group("/files")
		{
			files.POST("", m.handleUpload)
			files.POST("/batch", m.handleBatchUpload)
			files.GET("", m.handleList)
			files.GET("/:id", m.handleGet)
			files.GET("/:id/info", m.handleGetInfo)
			files.DELETE("/:id", m.handleDelete)
		}
	}
}

// Handlers
func (m *Module) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func (m *Module) handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"filename": file.Filename,
		"size":     file.Size,
	})
}

func (m *Module) handleBatchUpload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	files := form.File["files"]
	c.JSON(http.StatusCreated, gin.H{"count": len(files)})
}

func (m *Module) handleList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"files": []any{}})
}

func (m *Module) handleGet(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (m *Module) handleGetInfo(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "info": true})
}

func (m *Module) handleDelete(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Middleware
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
