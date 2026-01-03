package task

import (
	"context"
	"time"
)

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

// TaskPort defines the interface for task operations (hexagonal port).
// This is the contract that driving adapters (like HTTP API) use to interact
// with the core domain.
type TaskPort interface {
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error)
	GetTask(ctx context.Context, taskID string) (*TaskResponse, error)
	UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*TaskResponse, error)
	DeleteTask(ctx context.Context, taskID, userID string) error
	ListTasks(ctx context.Context, userID string) (*ListTasksResponse, error)
	CompleteTask(ctx context.Context, taskID string) (*TaskResponse, error)
}
