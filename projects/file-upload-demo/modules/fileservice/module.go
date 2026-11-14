// Package fileservice is the file-service module: file storage on the
// fs-jetstream plugin and the Gin HTTP API that drives it.
package fileservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
)

// Module implements the file service module using the fs-jetstream plugin.
// It owns both the storage service (service.go) and the HTTP API (http.go,
// handlers.go) that uses it in-process. It provides no services to other
// modules, so it has no contract package.
type Module struct {
	storage       *fsjetstream.PluginModule
	service       *Service
	server        *http.Server
	port          int
	maxUploadSize int64
	logger        types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module          = (*Module)(nil)
	_ mono.UsePluginModule = (*Module)(nil)
)

// NewModule creates a new file service module serving its HTTP API on port.
func NewModule(port int, maxUploadSize int64, logger types.Logger) *Module {
	return &Module{
		port:          port,
		maxUploadSize: maxUploadSize,
		logger:        logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "file-service"
}

// SetPlugin receives the storage plugin from the framework.
// This is called before Start() when the module implements UsePluginModule.
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "storage" {
		storage, ok := plugin.(*fsjetstream.PluginModule)
		if !ok {
			m.logger.Error("Invalid plugin type for storage",
				"alias", alias,
				"expected", "*fsjetstream.PluginModule")
			return
		}
		m.storage = storage
		m.logger.Info("Received storage plugin", "alias", alias)
	}
}

// Start initializes the storage service, then starts the HTTP server.
func (m *Module) Start(_ context.Context) error {
	if m.storage == nil {
		return fmt.Errorf("required plugin 'storage' not registered")
	}

	// Get the files bucket from the storage plugin
	bucket := m.storage.Bucket("files")
	if bucket == nil {
		return fmt.Errorf("bucket 'files' not found in storage plugin")
	}

	// Create the service
	m.service = NewService(bucket)

	if err := m.startHTTPServer(); err != nil {
		return err
	}

	m.logger.Info("File service module started")
	return nil
}

// Stop gracefully shuts down the HTTP server; in-flight uploads and
// downloads finish (within ctx) before the storage plugin is stopped.
func (m *Module) Stop(ctx context.Context) error {
	if err := m.stopHTTPServer(ctx); err != nil {
		return fmt.Errorf("failed to shut down HTTP server: %w", err)
	}
	m.logger.Info("File service module stopped")
	return nil
}
