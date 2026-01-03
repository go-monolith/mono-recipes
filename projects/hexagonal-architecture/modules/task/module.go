package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/example/hexagonal-architecture/events"
	"github.com/example/hexagonal-architecture/modules/user"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// TaskModule provides task management services (core domain).
type TaskModule struct {
	repo     *TaskRepository
	userPort user.UserPort
	eventBus mono.EventBus
}

// Compile-time interface checks.
var _ mono.Module = (*TaskModule)(nil)
var _ mono.ServiceProviderModule = (*TaskModule)(nil)
var _ mono.DependentModule = (*TaskModule)(nil)
var _ mono.EventEmitterModule = (*TaskModule)(nil)

// NewModule creates a new TaskModule.
func NewModule() *TaskModule {
	return &TaskModule{
		repo: NewTaskRepository(),
	}
}

// Name returns the module name.
func (m *TaskModule) Name() string {
	return "task"
}

// Dependencies returns the list of module dependencies.
// The framework will call SetDependencyServiceContainer for each dependency.
func (m *TaskModule) Dependencies() []string {
	return []string{"user"}
}

// SetDependencyServiceContainer receives service containers from dependencies.
// This is called by the framework for each dependency declared in Dependencies().
func (m *TaskModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	switch dependency {
	case "user":
		m.userPort = user.NewUserAdapter(container)
	}
}

// SetEventBus is called by the framework to inject the event bus.
func (m *TaskModule) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents returns all event definitions this module can emit.
// This implements the EventEmitterModule interface for event discovery.
func (m *TaskModule) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		events.TaskCreatedV1.ToBase(),
		events.TaskCompletedV1.ToBase(),
		events.TaskDeletedV1.ToBase(),
	}
}

// RegisterServices registers request-reply services in the service container.
// This implements the ServiceProviderModule interface.
func (m *TaskModule) RegisterServices(container mono.ServiceContainer) error {
	// Register create-task service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"create-task",
		json.Unmarshal,
		json.Marshal,
		m.createTask,
	); err != nil {
		return fmt.Errorf("failed to register create-task service: %w", err)
	}

	// Register get-task service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"get-task",
		json.Unmarshal,
		json.Marshal,
		m.getTask,
	); err != nil {
		return fmt.Errorf("failed to register get-task service: %w", err)
	}

	// Register update-task service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"update-task",
		json.Unmarshal,
		json.Marshal,
		m.updateTask,
	); err != nil {
		return fmt.Errorf("failed to register update-task service: %w", err)
	}

	// Register delete-task service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"delete-task",
		json.Unmarshal,
		json.Marshal,
		m.deleteTask,
	); err != nil {
		return fmt.Errorf("failed to register delete-task service: %w", err)
	}

	// Register list-tasks service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"list-tasks",
		json.Unmarshal,
		json.Marshal,
		m.listTasks,
	); err != nil {
		return fmt.Errorf("failed to register list-tasks service: %w", err)
	}

	// Register complete-task service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"complete-task",
		json.Unmarshal,
		json.Marshal,
		m.completeTask,
	); err != nil {
		return fmt.Errorf("failed to register complete-task service: %w", err)
	}

	log.Printf("[task] Registered services: create-task, get-task, update-task, delete-task, list-tasks, complete-task")
	return nil
}

// Start initializes the module.
func (m *TaskModule) Start(_ context.Context) error {
	if m.userPort == nil {
		return fmt.Errorf("userPort dependency not set")
	}
	if m.eventBus == nil {
		log.Println("[task] Warning: eventBus not set, events will not be published")
	}
	log.Println("[task] Module started (depends on: user)")
	return nil
}

// Stop shuts down the module.
func (m *TaskModule) Stop(_ context.Context) error {
	log.Println("[task] Module stopped")
	return nil
}
