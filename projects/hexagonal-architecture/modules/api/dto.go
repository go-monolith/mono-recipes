package api

import "time"

// CreateTaskRequest is the HTTP request for creating a task.
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	UserID      string `json:"user_id"`
}

// UpdateTaskRequest is the HTTP request for updating a task.
type UpdateTaskRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// TaskResponse is the HTTP response for a single task.
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

// ListTasksResponse is the HTTP response for listing tasks.
type ListTasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// HealthResponse is the HTTP response for health check.
type HealthResponse struct {
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}

// ErrorResponse is the HTTP response for errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
