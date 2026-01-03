package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/example/file-upload-demo/modules/files"
	"github.com/gin-gonic/gin"
	"github.com/go-monolith/mono"
)

// APIModule is the driving adapter that exposes REST endpoints using Gin.
type APIModule struct {
	router       *gin.Engine
	server       *http.Server
	filesAdapter files.FilesPort
	port         string
}

// Compile-time interface checks.
var _ mono.Module = (*APIModule)(nil)
var _ mono.DependentModule = (*APIModule)(nil)
var _ mono.HealthCheckableModule = (*APIModule)(nil)

// NewModule creates a new APIModule.
func NewModule() *APIModule {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return &APIModule{
		port: port,
	}
}

// Name returns the module name.
func (m *APIModule) Name() string {
	return "api"
}

// Dependencies returns the list of module dependencies.
func (m *APIModule) Dependencies() []string {
	return []string{"files"}
}

// SetDependencyServiceContainer receives service containers from dependencies.
func (m *APIModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	switch dependency {
	case "files":
		m.filesAdapter = files.NewFilesAdapter(container)
	}
}

// Start initializes the Gin HTTP server.
func (m *APIModule) Start(_ context.Context) error {
	if m.filesAdapter == nil {
		return fmt.Errorf("filesAdapter dependency not set")
	}

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create router with default middleware
	m.router = gin.New()
	m.router.Use(gin.Recovery())
	m.router.Use(loggerMiddleware())

	// Setup routes
	m.setupRoutes()

	// Create HTTP server with comprehensive timeouts to prevent DoS attacks
	m.server = &http.Server{
		Addr:              ":" + m.port,
		Handler:           m.router,
		ReadTimeout:       30 * time.Second, // Time to read request body
		ReadHeaderTimeout: 10 * time.Second, // Time to read request headers
		WriteTimeout:      60 * time.Second, // Time to write response (generous for downloads)
		IdleTimeout:       120 * time.Second, // Keep-alive timeout
	}

	// Start server in goroutine
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[api] HTTP server error: %v", err)
		}
	}()

	log.Printf("[api] HTTP server started on :%s", m.port)
	return nil
}

// Stop shuts down the Gin HTTP server.
func (m *APIModule) Stop(ctx context.Context) error {
	if m.server == nil {
		return nil
	}
	log.Println("[api] Shutting down HTTP server...")
	return m.server.Shutdown(ctx)
}

// Health returns the health status of the module.
func (m *APIModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: m.server != nil,
		Message: "operational",
		Details: map[string]any{
			"port": m.port,
		},
	}
}

// loggerMiddleware returns a Gin middleware for request logging.
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[api] %s %s %d %v", c.Request.Method, path, status, latency)
	}
}
