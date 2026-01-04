package fileservice

import (
	"context"
	"fmt"

	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
	"github.com/go-monolith/mono/pkg/types"
)

// Module implements the file service module using the fs-jetstream plugin.
type Module struct {
	storage *fsjetstream.PluginModule
	bucket  fsjetstream.FileStoragePort
	service *Service
	logger  types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module          = (*Module)(nil)
	_ mono.UsePluginModule = (*Module)(nil)
)

// NewModule creates a new file service module.
func NewModule(logger types.Logger) *Module {
	return &Module{
		logger: logger,
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

// Start initializes the module and its service.
func (m *Module) Start(ctx context.Context) error {
	if m.storage == nil {
		return fmt.Errorf("required plugin 'storage' not registered")
	}

	// Get the files bucket from the storage plugin
	m.bucket = m.storage.Bucket("files")
	if m.bucket == nil {
		return fmt.Errorf("bucket 'files' not found in storage plugin")
	}

	// Create the service
	m.service = NewService(m.bucket)

	m.logger.Info("File service module started")
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("File service module stopped")
	return nil
}

// Service returns the file service instance.
func (m *Module) Service() *Service {
	return m.service
}
