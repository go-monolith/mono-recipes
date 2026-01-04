package worker

import (
	"context"
	"log"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/eventbus"
	"github.com/example/background-jobs-demo/modules/nats"
	"github.com/go-monolith/mono"
)

// Module provides the worker pool as a mono module.
type Module struct {
	pool       *Pool
	config     PoolConfig
	natsClient *nats.Client
	eventBus   *eventbus.EventBus
	jobStore   *job.Store
}

// NewModule creates a new worker module with default configuration.
func NewModule(natsClient *nats.Client, eventBus *eventbus.EventBus, jobStore *job.Store) *Module {
	return &Module{
		config:     DefaultPoolConfig(),
		natsClient: natsClient,
		eventBus:   eventBus,
		jobStore:   jobStore,
	}
}

// NewModuleWithConfig creates a new worker module with custom configuration.
func NewModuleWithConfig(cfg PoolConfig, natsClient *nats.Client, eventBus *eventbus.EventBus, jobStore *job.Store) *Module {
	return &Module{
		config:     cfg,
		natsClient: natsClient,
		eventBus:   eventBus,
		jobStore:   jobStore,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "worker"
}

// Init initializes the worker pool.
func (m *Module) Init(_ mono.ServiceContainer) error {
	m.pool = NewPool(m.config, m.natsClient, m.eventBus, m.jobStore)
	log.Println("[worker] Worker pool initialized")
	return nil
}

// Start starts the worker pool.
func (m *Module) Start(ctx context.Context) error {
	if err := m.pool.Start(ctx); err != nil {
		return err
	}
	log.Println("[worker] Module started")
	return nil
}

// Stop stops the worker pool gracefully.
func (m *Module) Stop(ctx context.Context) error {
	if err := m.pool.Stop(ctx); err != nil {
		return err
	}
	log.Println("[worker] Module stopped")
	return nil
}

// GetPool returns the worker pool instance.
func (m *Module) GetPool() *Pool {
	return m.pool
}
