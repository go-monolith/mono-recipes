package task

import (
	"fmt"
	"sync"

	domain "github.com/example/hexagonal-architecture/domain/task"
)

// TaskRepository provides in-memory task storage.
type TaskRepository struct {
	tasks map[string]*domain.Task
	mu    sync.RWMutex
}

// NewTaskRepository creates a new task repository.
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[string]*domain.Task),
	}
}

// Save saves a task to the repository.
func (r *TaskRepository) Save(task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks[task.ID] = task
	return nil
}

// FindByID finds a task by ID.
func (r *TaskRepository) FindByID(taskID string) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, found := r.tasks[taskID]
	if !found {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// Delete deletes a task by ID.
func (r *TaskRepository) Delete(taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.tasks[taskID]; !found {
		return fmt.Errorf("task not found: %s", taskID)
	}
	delete(r.tasks, taskID)
	return nil
}

// FindByUserID finds all tasks for a user.
func (r *TaskRepository) FindByUserID(userID string) []*domain.Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Task
	for _, task := range r.tasks {
		if userID == "" || task.UserID == userID {
			result = append(result, task)
		}
	}
	return result
}

// FindAll returns all tasks.
func (r *TaskRepository) FindAll() []*domain.Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		result = append(result, task)
	}
	return result
}
