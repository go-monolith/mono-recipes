// Package contract is the published port of the task module (the core
// domain): the DTOs of its services, the typed contracts driving adapters call
// them through, and the events the module emits. Event consumers import the
// event definitions from here, so no shared "events" package is needed.
//
// A contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// ---------------------------------------------------------------------------
// Service DTOs
// ---------------------------------------------------------------------------

// CreateTaskRequest is the request for creating a task.
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	UserID      string `json:"user_id"`
}

// CreateTaskResponse is the response for creating a task.
type CreateTaskResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GetTaskRequest is the request for getting a task.
type GetTaskRequest struct {
	TaskID string `json:"task_id"`
}

// UpdateTaskRequest is the request for updating a task.
type UpdateTaskRequest struct {
	TaskID      string `json:"task_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// DeleteTaskRequest is the request for deleting a task.
type DeleteTaskRequest struct {
	TaskID string `json:"task_id"`
	UserID string `json:"user_id"`
}

// DeleteTaskResponse is the response for deleting a task.
type DeleteTaskResponse struct {
	Deleted bool `json:"deleted"`
}

// ListTasksRequest is the request for listing tasks.
type ListTasksRequest struct {
	UserID string `json:"user_id,omitempty"`
}

// ListTasksResponse is the response for listing tasks.
type ListTasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// CompleteTaskRequest is the request for completing a task.
type CompleteTaskRequest struct {
	TaskID string `json:"task_id"`
}

// TaskResponse is the response for a single task.
type TaskResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	UserID      string     `json:"user_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Service contracts (subjects: services.task.<name>)
// ---------------------------------------------------------------------------

// CreateTask creates a task for an existing user and emits TaskCreatedV1.
var CreateTask = helper.RequestReplyService[CreateTaskRequest, CreateTaskResponse]("task", "create-task")

// GetTask returns a task by ID.
var GetTask = helper.RequestReplyService[GetTaskRequest, TaskResponse]("task", "get-task")

// UpdateTask updates the title and/or description of a task.
var UpdateTask = helper.RequestReplyService[UpdateTaskRequest, TaskResponse]("task", "update-task")

// DeleteTask deletes a task and emits TaskDeletedV1.
var DeleteTask = helper.RequestReplyService[DeleteTaskRequest, DeleteTaskResponse]("task", "delete-task")

// ListTasks lists all tasks, optionally filtered by user.
var ListTasks = helper.RequestReplyService[ListTasksRequest, ListTasksResponse]("task", "list-tasks")

// CompleteTask marks a task as completed and emits TaskCompletedV1.
var CompleteTask = helper.RequestReplyService[CompleteTaskRequest, TaskResponse]("task", "complete-task")

// ---------------------------------------------------------------------------
// Events emitted by the task module
// ---------------------------------------------------------------------------

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
var TaskCreatedV1 = helper.EventDefinition[TaskCreatedEvent]("task", "TaskCreated", "v1")

// TaskCompletedEvent is emitted when a task is marked complete.
type TaskCompletedEvent struct {
	TaskID      string    `json:"task_id"`
	UserID      string    `json:"user_id"`
	CompletedAt time.Time `json:"completed_at"`
}

// TaskCompletedV1 is the typed event definition for task completion.
// Subject: events.task.v1.task-completed
var TaskCompletedV1 = helper.EventDefinition[TaskCompletedEvent]("task", "TaskCompleted", "v1")

// TaskDeletedEvent is emitted when a task is deleted.
type TaskDeletedEvent struct {
	TaskID    string    `json:"task_id"`
	UserID    string    `json:"user_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// TaskDeletedV1 is the typed event definition for task deletion.
// Subject: events.task.v1.task-deleted
var TaskDeletedV1 = helper.EventDefinition[TaskDeletedEvent]("task", "TaskDeleted", "v1")
