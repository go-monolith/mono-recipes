// Package task is the core domain of the demo: task management services
// declared in task/contract, plus the TaskCreated/TaskCompleted/TaskDeleted
// events. It depends on the user module only through user/contract and the
// UserPort it owns.
package task

import (
	"context"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/hexagonal-architecture/modules/task/contract"
	usercontract "github.com/example/hexagonal-architecture/modules/user/contract"
)

// TaskModule provides task management services (core domain).
type TaskModule struct {
	repo     *TaskRepository
	userPort UserPort
	eventBus mono.EventBus
}

var (
	_ mono.Module                 = (*TaskModule)(nil)
	_ mono.ServiceProviderModule  = (*TaskModule)(nil)
	_ mono.DependentModule        = (*TaskModule)(nil)
	_ mono.EventEmitterModule     = (*TaskModule)(nil)
	_ types.ServiceConsumerModule = (*TaskModule)(nil) // no mono.* alias in v0.1.1
)

// NewModule creates the task module.
func NewModule() *TaskModule {
	return &TaskModule{
		repo: NewTaskRepository(),
	}
}

func (m *TaskModule) Name() string {
	return "task"
}

func (m *TaskModule) Dependencies() []string {
	return []string{"user"}
}

// ConsumeServices declares every contract the task module calls. The framework
// validates the list at startup against what the user module registered.
func (m *TaskModule) ConsumeServices() []types.BaseServiceDefinition {
	return []types.BaseServiceDefinition{
		usercontract.ValidateUser.ToBase(),
	}
}

// SetDependencyServiceContainer wires the adapter behind UserPort. The
// container is a read-only view of the user module's services.
func (m *TaskModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == "user" {
		m.userPort = userAdapter{container: container}
	}
}

func (m *TaskModule) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

func (m *TaskModule) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		contract.TaskCreatedV1.ToBase(),
		contract.TaskCompletedV1.ToBase(),
		contract.TaskDeletedV1.ToBase(),
	}
}

// RegisterServices is the driving adapter: it binds the domain handlers in
// service.go to the module's own contracts.
func (m *TaskModule) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.CreateTask.Register(container, m.createTask); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.CreateTask.Name, err)
	}
	if err := contract.GetTask.Register(container, m.getTask); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetTask.Name, err)
	}
	if err := contract.UpdateTask.Register(container, m.updateTask); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.UpdateTask.Name, err)
	}
	if err := contract.DeleteTask.Register(container, m.deleteTask); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.DeleteTask.Name, err)
	}
	if err := contract.ListTasks.Register(container, m.listTasks); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.ListTasks.Name, err)
	}
	if err := contract.CompleteTask.Register(container, m.completeTask); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.CompleteTask.Name, err)
	}

	log.Printf("[task] Registered services: %s, %s, %s, %s, %s, %s",
		contract.CreateTask.Name, contract.GetTask.Name, contract.UpdateTask.Name,
		contract.DeleteTask.Name, contract.ListTasks.Name, contract.CompleteTask.Name)
	return nil
}

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

func (m *TaskModule) Stop(_ context.Context) error {
	log.Println("[task] Module stopped")
	return nil
}
