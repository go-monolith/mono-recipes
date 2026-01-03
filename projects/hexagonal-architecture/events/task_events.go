package events

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// TaskCreatedEvent is emitted when a new task is created.
type TaskCreatedEvent struct {
	TaskID      string    `json:"task_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskCreatedV1 is the typed event definition for task creation.
// Subject: events.task.v1.task-created
var TaskCreatedV1 = helper.EventDefinition[TaskCreatedEvent](
	"task", "TaskCreated", "v1",
)

// TaskCompletedEvent is emitted when a task is marked complete.
type TaskCompletedEvent struct {
	TaskID      string    `json:"task_id"`
	UserID      string    `json:"user_id"`
	CompletedAt time.Time `json:"completed_at"`
}

// TaskCompletedV1 is the typed event definition for task completion.
// Subject: events.task.v1.task-completed
var TaskCompletedV1 = helper.EventDefinition[TaskCompletedEvent](
	"task", "TaskCompleted", "v1",
)

// TaskDeletedEvent is emitted when a task is deleted.
type TaskDeletedEvent struct {
	TaskID    string    `json:"task_id"`
	UserID    string    `json:"user_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// TaskDeletedV1 is the typed event definition for task deletion.
// Subject: events.task.v1.task-deleted
var TaskDeletedV1 = helper.EventDefinition[TaskDeletedEvent](
	"task", "TaskDeleted", "v1",
)
