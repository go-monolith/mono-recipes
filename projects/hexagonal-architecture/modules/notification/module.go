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

var _ mono.Module = (*NotificationModule)(nil)
var _ mono.EventConsumerModule = (*NotificationModule)(nil)

func NewModule() *NotificationModule {
	return &NotificationModule{
		notifications: make([]NotificationLog, 0),
	}
}

func (m *NotificationModule) Name() string {
	return "notification"
}

func (m *NotificationModule) RegisterEventConsumers(registry mono.EventRegistry) error {
	if err := helper.RegisterTypedEventConsumer(registry, events.TaskCreatedV1, m.handleTaskCreated, m); err != nil {
		return fmt.Errorf("failed to register TaskCreated consumer: %w", err)
	}
	if err := helper.RegisterTypedEventConsumer(registry, events.TaskCompletedV1, m.handleTaskCompleted, m); err != nil {
		return fmt.Errorf("failed to register TaskCompleted consumer: %w", err)
	}
	if err := helper.RegisterTypedEventConsumer(registry, events.TaskDeletedV1, m.handleTaskDeleted, m); err != nil {
		return fmt.Errorf("failed to register TaskDeleted consumer: %w", err)
	}

	log.Printf("[notification] Registered event consumers: TaskCreated, TaskCompleted, TaskDeleted")
	return nil
}

func (m *NotificationModule) handleTaskCreated(_ context.Context, event events.TaskCreatedEvent, _ *mono.Msg) error {
	log.Printf("[notification] Task created: %s - %s", event.TaskID, event.Title)
	m.logNotification(event.TaskID, "task_created", fmt.Sprintf("New task '%s' created for user %s", event.Title, event.UserID))
	return nil
}

func (m *NotificationModule) handleTaskCompleted(_ context.Context, event events.TaskCompletedEvent, _ *mono.Msg) error {
	log.Printf("[notification] Task completed: %s by user %s", event.TaskID, event.UserID)
	m.logNotification(event.TaskID, "task_completed", fmt.Sprintf("Task %s completed!", event.TaskID))
	return nil
}

func (m *NotificationModule) handleTaskDeleted(_ context.Context, event events.TaskDeletedEvent, _ *mono.Msg) error {
	log.Printf("[notification] Task deleted: %s by user %s", event.TaskID, event.UserID)
	m.logNotification(event.TaskID, "task_deleted", fmt.Sprintf("Task %s deleted", event.TaskID))
	return nil
}

func (m *NotificationModule) logNotification(id, notificationType, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, NotificationLog{
		ID:        id,
		Type:      notificationType,
		Message:   message,
		Channel:   "event",
		Timestamp: time.Now(),
	})
}

func (m *NotificationModule) GetNotifications() []NotificationLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NotificationLog, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func (m *NotificationModule) Start(_ context.Context) error {
	log.Println("[notification] Module started - listening for task events")
	return nil
}

func (m *NotificationModule) Stop(_ context.Context) error {
	log.Println("[notification] Module stopped")
	return nil
}
