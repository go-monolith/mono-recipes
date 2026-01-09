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

var _ mono.Module = (*TaskModule)(nil)
var _ mono.ServiceProviderModule = (*TaskModule)(nil)
var _ mono.DependentModule = (*TaskModule)(nil)
var _ mono.EventEmitterModule = (*TaskModule)(nil)

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

func (m *TaskModule) SetDependencyServiceContainer(dependency string, container mono.ServiceContainer) {
	if dependency == "user" {
		m.userPort = user.NewUserAdapter(container)
	}
}

func (m *TaskModule) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

func (m *TaskModule) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		events.TaskCreatedV1.ToBase(),
		events.TaskCompletedV1.ToBase(),
		events.TaskDeletedV1.ToBase(),
	}
}

func (m *TaskModule) RegisterServices(container mono.ServiceContainer) error {
	if err := helper.RegisterTypedRequestReplyService(
		container, "create-task", json.Unmarshal, json.Marshal, m.createTask,
	); err != nil {
		return fmt.Errorf("failed to register create-task service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "get-task", json.Unmarshal, json.Marshal, m.getTask,
	); err != nil {
		return fmt.Errorf("failed to register get-task service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "update-task", json.Unmarshal, json.Marshal, m.updateTask,
	); err != nil {
		return fmt.Errorf("failed to register update-task service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "delete-task", json.Unmarshal, json.Marshal, m.deleteTask,
	); err != nil {
		return fmt.Errorf("failed to register delete-task service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "list-tasks", json.Unmarshal, json.Marshal, m.listTasks,
	); err != nil {
		return fmt.Errorf("failed to register list-tasks service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "complete-task", json.Unmarshal, json.Marshal, m.completeTask,
	); err != nil {
		return fmt.Errorf("failed to register complete-task service: %w", err)
	}

	log.Printf("[task] Registered services: create-task, get-task, update-task, delete-task, list-tasks, complete-task")
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
