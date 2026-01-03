package task

import "time"

// TaskStatus represents the state of a task.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusCompleted TaskStatus = "completed"
)

// Task is the core domain entity representing a todo item.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	UserID      string     `json:"user_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
