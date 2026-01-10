package fileops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"
	"github.com/go-monolith/mono/pkg/helper"
	"github.com/go-monolith/mono/pkg/types"
)

// Module implements the file operations module using the fs-jetstream plugin.
type Module struct {
	storage *fsjetstream.PluginModule
	bucket  fsjetstream.FileStoragePort
	logger  types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.UsePluginModule       = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new file operations module.
func NewModule(logger types.Logger) *Module {
	return &Module{
		logger: logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "fileops"
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

// Start initializes the module.
func (m *Module) Start(ctx context.Context) error {
	if m.storage == nil {
		return fmt.Errorf("required plugin 'storage' not registered")
	}

	// Get the user-settings bucket from the storage plugin
	m.bucket = m.storage.Bucket("user-settings")
	if m.bucket == nil {
		return fmt.Errorf("bucket 'user-settings' not found in storage plugin")
	}

	m.logger.Info("File operations module started")
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("File operations module stopped")
	return nil
}

// RegisterServices registers the file operations services with the service container.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// Register RequestReplyService for file.save
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"save",
		json.Unmarshal,
		json.Marshal,
		m.handleFileSave,
	); err != nil {
		return fmt.Errorf("failed to register save service: %w", err)
	}

	// Register QueueGroupService for file.archive
	if err := container.RegisterQueueGroupService("archive", mono.QGHP{
		QueueGroup: "archive-workers",
		Handler:    m.handleFileArchive,
	}); err != nil {
		return fmt.Errorf("failed to register archive service: %w", err)
	}

	m.logger.Info("Registered services",
		"services", "services.fileops.save, services.fileops.archive")
	return nil
}
