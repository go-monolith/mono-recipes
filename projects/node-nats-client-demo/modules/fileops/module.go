package fileops

import (
	"context"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	fsjetstream "github.com/go-monolith/mono/plugin/fs-jetstream"

	"node-nats-client-demo/modules/fileops/contract"
)

// archiveQueueGroup is the queue group of the archive workers. It is a
// provider-side detail: senders only need the contract's subject.
const archiveQueueGroup = "archive-workers"

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

// RegisterServices binds the typed handlers to the module's own contracts,
// which fix the subjects and JSON encoding the Node.js clients rely on.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// RequestReplyService on services.fileops.save
	if err := contract.Save.Register(container, m.handleFileSave); err != nil {
		return fmt.Errorf("failed to register save service: %w", err)
	}

	// QueueGroupService on services.fileops.archive
	if err := contract.Archive.Register(container, types.TypedQGHP[contract.FileArchiveRequest]{
		QueueGroup: archiveQueueGroup,
		Handler:    m.handleFileArchive,
	}); err != nil {
		return fmt.Errorf("failed to register archive service: %w", err)
	}

	m.logger.Info("Registered services",
		"services", []string{contract.Save.Subject(), contract.Archive.Subject()})
	return nil
}
