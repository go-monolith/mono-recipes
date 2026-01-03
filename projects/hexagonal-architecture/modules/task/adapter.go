package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// taskAdapter wraps ServiceContainer for type-safe cross-module communication.
// This is the adapter that implements the TaskPort interface.
type taskAdapter struct {
	container mono.ServiceContainer
}

// NewTaskAdapter creates a new adapter for task services.
// container is the ServiceContainer from the task module received via SetDependencyServiceContainer.
func NewTaskAdapter(container mono.ServiceContainer) TaskPort {
	if container == nil {
		panic("task adapter requires non-nil ServiceContainer")
	}
	return &taskAdapter{container: container}
}

// CreateTask creates a new task via the create-task service.
func (a *taskAdapter) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	var resp CreateTaskResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"create-task",
		json.Marshal,
		json.Unmarshal,
		req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("create-task service call failed: %w", err)
	}
	return &resp, nil
}

// GetTask retrieves a task by ID via the get-task service.
func (a *taskAdapter) GetTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	req := GetTaskRequest{TaskID: taskID}
	var resp TaskResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"get-task",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("get-task service call failed: %w", err)
	}
	return &resp, nil
}

// UpdateTask updates a task via the update-task service.
func (a *taskAdapter) UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*TaskResponse, error) {
	var resp TaskResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"update-task",
		json.Marshal,
		json.Unmarshal,
		req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("update-task service call failed: %w", err)
	}
	return &resp, nil
}

// DeleteTask deletes a task via the delete-task service.
func (a *taskAdapter) DeleteTask(ctx context.Context, taskID, userID string) error {
	req := DeleteTaskRequest{TaskID: taskID, UserID: userID}
	var resp DeleteTaskResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"delete-task",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("delete-task service call failed: %w", err)
	}
	if !resp.Deleted {
		return fmt.Errorf("task not deleted: %s", taskID)
	}
	return nil
}

// ListTasks lists all tasks, optionally filtered by user, via the list-tasks service.
func (a *taskAdapter) ListTasks(ctx context.Context, userID string) (*ListTasksResponse, error) {
	req := ListTasksRequest{UserID: userID}
	var resp ListTasksResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"list-tasks",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("list-tasks service call failed: %w", err)
	}
	return &resp, nil
}

// CompleteTask marks a task as completed via the complete-task service.
func (a *taskAdapter) CompleteTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	req := CompleteTaskRequest{TaskID: taskID}
	var resp TaskResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"complete-task",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("complete-task service call failed: %w", err)
	}
	return &resp, nil
}
