package task

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono"
	"github.com/google/uuid"

	domain "github.com/example/hexagonal-architecture/domain/task"
	"github.com/example/hexagonal-architecture/modules/task/contract"
	usercontract "github.com/example/hexagonal-architecture/modules/user/contract"
)

// createTask handles the create-task service request.
func (m *TaskModule) createTask(ctx context.Context, req contract.CreateTaskRequest, _ *mono.Msg) (contract.CreateTaskResponse, error) {
	// Validate user exists via the user port (driven adapter pattern)
	user, err := m.userPort.ValidateUser(ctx, usercontract.ValidateUserRequest{UserID: req.UserID})
	if err != nil {
		return contract.CreateTaskResponse{}, fmt.Errorf("failed to validate user: %w", err)
	}
	if !user.Valid {
		return contract.CreateTaskResponse{}, fmt.Errorf("invalid user: %s", req.UserID)
	}

	// Create domain entity
	now := time.Now()
	newTask := &domain.Task{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Status:      domain.StatusPending,
		UserID:      req.UserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to repository
	if err := m.repo.Save(newTask); err != nil {
		return contract.CreateTaskResponse{}, fmt.Errorf("failed to save task: %w", err)
	}

	// Emit TaskCreated event using typed Publish method
	if m.eventBus != nil {
		event := contract.TaskCreatedEvent{
			TaskID:      newTask.ID,
			Title:       newTask.Title,
			Description: newTask.Description,
			UserID:      newTask.UserID,
			CreatedAt:   newTask.CreatedAt,
		}
		if err := contract.TaskCreatedV1.Publish(m.eventBus, event, nil); err != nil {
			// Event publishing is best-effort; log but don't fail the operation
			log.Printf("[task] Warning: failed to publish TaskCreated event for task %s: %v", newTask.ID, err)
		}
	}

	return contract.CreateTaskResponse{
		ID:        newTask.ID,
		Title:     newTask.Title,
		Status:    string(newTask.Status),
		CreatedAt: newTask.CreatedAt,
	}, nil
}

// getTask handles the get-task service request.
func (m *TaskModule) getTask(_ context.Context, req contract.GetTaskRequest, _ *mono.Msg) (contract.TaskResponse, error) {
	task, err := m.repo.FindByID(req.TaskID)
	if err != nil {
		return contract.TaskResponse{}, err
	}
	return toTaskResponse(task), nil
}

// updateTask handles the update-task service request.
func (m *TaskModule) updateTask(_ context.Context, req contract.UpdateTaskRequest, _ *mono.Msg) (contract.TaskResponse, error) {
	task, err := m.repo.FindByID(req.TaskID)
	if err != nil {
		return contract.TaskResponse{}, err
	}

	// Update fields if provided
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	task.UpdatedAt = time.Now()

	if err := m.repo.Save(task); err != nil {
		return contract.TaskResponse{}, fmt.Errorf("failed to update task: %w", err)
	}

	return toTaskResponse(task), nil
}

// deleteTask handles the delete-task service request.
func (m *TaskModule) deleteTask(_ context.Context, req contract.DeleteTaskRequest, _ *mono.Msg) (contract.DeleteTaskResponse, error) {
	task, err := m.repo.FindByID(req.TaskID)
	if err != nil {
		return contract.DeleteTaskResponse{Deleted: false}, err
	}

	if err := m.repo.Delete(req.TaskID); err != nil {
		return contract.DeleteTaskResponse{Deleted: false}, err
	}

	// Emit TaskDeleted event
	if m.eventBus != nil {
		event := contract.TaskDeletedEvent{
			TaskID:    req.TaskID,
			UserID:    task.UserID,
			DeletedAt: time.Now(),
		}
		if err := contract.TaskDeletedV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[task] Warning: failed to publish TaskDeleted event for task %s: %v", req.TaskID, err)
		}
	}

	return contract.DeleteTaskResponse{Deleted: true}, nil
}

// listTasks handles the list-tasks service request.
func (m *TaskModule) listTasks(_ context.Context, req contract.ListTasksRequest, _ *mono.Msg) (contract.ListTasksResponse, error) {
	var tasks []*domain.Task
	if req.UserID != "" {
		tasks = m.repo.FindByUserID(req.UserID)
	} else {
		tasks = m.repo.FindAll()
	}

	response := contract.ListTasksResponse{
		Tasks: make([]contract.TaskResponse, 0, len(tasks)),
		Total: len(tasks),
	}

	for _, task := range tasks {
		response.Tasks = append(response.Tasks, toTaskResponse(task))
	}

	return response, nil
}

// completeTask handles the complete-task service request.
func (m *TaskModule) completeTask(_ context.Context, req contract.CompleteTaskRequest, _ *mono.Msg) (contract.TaskResponse, error) {
	task, err := m.repo.FindByID(req.TaskID)
	if err != nil {
		return contract.TaskResponse{}, err
	}

	// Update status
	now := time.Now()
	task.Status = domain.StatusCompleted
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := m.repo.Save(task); err != nil {
		return contract.TaskResponse{}, fmt.Errorf("failed to complete task: %w", err)
	}

	// Emit TaskCompleted event
	if m.eventBus != nil {
		event := contract.TaskCompletedEvent{
			TaskID:      task.ID,
			UserID:      task.UserID,
			CompletedAt: now,
		}
		if err := contract.TaskCompletedV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[task] Warning: failed to publish TaskCompleted event for task %s: %v", task.ID, err)
		}
	}

	return toTaskResponse(task), nil
}

// toTaskResponse converts a domain Task to a contract.TaskResponse.
func toTaskResponse(task *domain.Task) contract.TaskResponse {
	return contract.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		UserID:      task.UserID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		CompletedAt: task.CompletedAt,
	}
}
