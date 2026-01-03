package notification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/example/hexagonal-architecture/events"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// NotificationLog represents a logged notification.
type NotificationLog struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Channel   string    `json:"channel"`
	Timestamp time.Time `json:"timestamp"`
}

// NotificationModule handles notifications as a driven adapter.
// It subscribes to domain events using the EventConsumerModule interface.
type NotificationModule struct {
	notifications []NotificationLog
	mu            sync.RWMutex
}

// Compile-time interface checks.
var _ mono.Module = (*NotificationModule)(nil)
var _ mono.EventConsumerModule = (*NotificationModule)(nil)

// NewModule creates a new NotificationModule.
func NewModule() *NotificationModule {
	return &NotificationModule{
		notifications: make([]NotificationLog, 0),
	}
}

// Name returns the module name.
func (m *NotificationModule) Name() string {
	return "notification"
}

// RegisterEventConsumers registers event consumers for this module.
// Called by the framework after RegisterServices but before Start.
// This implements the EventConsumerModule interface.
func (m *NotificationModule) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Subscribe to TaskCreated events using type-safe RegisterTypedEventConsumer
	if err := helper.RegisterTypedEventConsumer(registry, events.TaskCreatedV1, m.handleTaskCreated, m); err != nil {
		return fmt.Errorf("failed to register TaskCreated consumer: %w", err)
	}

	// Subscribe to TaskCompleted events
	if err := helper.RegisterTypedEventConsumer(registry, events.TaskCompletedV1, m.handleTaskCompleted, m); err != nil {
		return fmt.Errorf("failed to register TaskCompleted consumer: %w", err)
	}

	// Subscribe to TaskDeleted events
	if err := helper.RegisterTypedEventConsumer(registry, events.TaskDeletedV1, m.handleTaskDeleted, m); err != nil {
		return fmt.Errorf("failed to register TaskDeleted consumer: %w", err)
	}

	log.Printf("[notification] Registered event consumers: TaskCreated, TaskCompleted, TaskDeleted")
	return nil
}

// handleTaskCreated handles TaskCreated events.
// This handler uses the TypedEventConsumerHandler signature with automatic unmarshaling.
func (m *NotificationModule) handleTaskCreated(_ context.Context, event events.TaskCreatedEvent, _ *mono.Msg) error {
	log.Printf("[notification] Task created: %s - %s", event.TaskID, event.Title)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, NotificationLog{
		ID:        event.TaskID,
		Type:      "task_created",
		Message:   fmt.Sprintf("New task '%s' created for user %s", event.Title, event.UserID),
		Channel:   "event",
		Timestamp: time.Now(),
	})

	// In a real system: send email, push notification, etc.
	return nil
}

// handleTaskCompleted handles TaskCompleted events.
func (m *NotificationModule) handleTaskCompleted(_ context.Context, event events.TaskCompletedEvent, _ *mono.Msg) error {
	log.Printf("[notification] Task completed: %s by user %s", event.TaskID, event.UserID)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, NotificationLog{
		ID:        event.TaskID,
		Type:      "task_completed",
		Message:   fmt.Sprintf("Task %s completed!", event.TaskID),
		Channel:   "event",
		Timestamp: time.Now(),
	})

	return nil
}

// handleTaskDeleted handles TaskDeleted events.
func (m *NotificationModule) handleTaskDeleted(_ context.Context, event events.TaskDeletedEvent, _ *mono.Msg) error {
	log.Printf("[notification] Task deleted: %s by user %s", event.TaskID, event.UserID)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, NotificationLog{
		ID:        event.TaskID,
		Type:      "task_deleted",
		Message:   fmt.Sprintf("Task %s deleted", event.TaskID),
		Channel:   "event",
		Timestamp: time.Now(),
	})

	return nil
}

// GetNotifications returns a copy of all logged notifications.
func (m *NotificationModule) GetNotifications() []NotificationLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NotificationLog, len(m.notifications))
	copy(result, m.notifications)
	return result
}

// Start initializes the module.
func (m *NotificationModule) Start(_ context.Context) error {
	log.Println("[notification] Module started - listening for task events")
	return nil
}

// Stop shuts down the module.
func (m *NotificationModule) Stop(_ context.Context) error {
	log.Println("[notification] Module stopped")
	return nil
}
