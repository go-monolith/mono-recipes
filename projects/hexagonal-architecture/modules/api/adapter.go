package api

import (
	"context"

	"github.com/go-monolith/mono"

	taskcontract "github.com/example/hexagonal-architecture/modules/task/contract"
)

// taskAdapter is the driven adapter behind TaskPort: every method is one call
// of a task contract through the container received in
// SetDependencyServiceContainer. Contract errors already carry the contract
// identity (e.g. "contract task/get-task (request_reply): ...").
type taskAdapter struct {
	container mono.ServiceContainer
}

var _ TaskPort = taskAdapter{}

func (a taskAdapter) CreateTask(ctx context.Context, req taskcontract.CreateTaskRequest) (taskcontract.CreateTaskResponse, error) {
	return taskcontract.CreateTask.Call(ctx, a.container, req)
}

func (a taskAdapter) GetTask(ctx context.Context, req taskcontract.GetTaskRequest) (taskcontract.TaskResponse, error) {
	return taskcontract.GetTask.Call(ctx, a.container, req)
}

func (a taskAdapter) UpdateTask(ctx context.Context, req taskcontract.UpdateTaskRequest) (taskcontract.TaskResponse, error) {
	return taskcontract.UpdateTask.Call(ctx, a.container, req)
}

func (a taskAdapter) DeleteTask(ctx context.Context, req taskcontract.DeleteTaskRequest) (taskcontract.DeleteTaskResponse, error) {
	return taskcontract.DeleteTask.Call(ctx, a.container, req)
}

func (a taskAdapter) ListTasks(ctx context.Context, req taskcontract.ListTasksRequest) (taskcontract.ListTasksResponse, error) {
	return taskcontract.ListTasks.Call(ctx, a.container, req)
}

func (a taskAdapter) CompleteTask(ctx context.Context, req taskcontract.CompleteTaskRequest) (taskcontract.TaskResponse, error) {
	return taskcontract.CompleteTask.Call(ctx, a.container, req)
}
