package api

import (
	"context"

	taskcontract "github.com/example/hexagonal-architecture/modules/task/contract"
)

// TaskPort is the driven port the HTTP driving adapter needs from the core
// domain. The api module owns it; it speaks the task contract's DTOs and says
// nothing about NATS, so handlers can be tested against a fake.
type TaskPort interface {
	CreateTask(ctx context.Context, req taskcontract.CreateTaskRequest) (taskcontract.CreateTaskResponse, error)
	GetTask(ctx context.Context, req taskcontract.GetTaskRequest) (taskcontract.TaskResponse, error)
	UpdateTask(ctx context.Context, req taskcontract.UpdateTaskRequest) (taskcontract.TaskResponse, error)
	DeleteTask(ctx context.Context, req taskcontract.DeleteTaskRequest) (taskcontract.DeleteTaskResponse, error)
	ListTasks(ctx context.Context, req taskcontract.ListTasksRequest) (taskcontract.ListTasksResponse, error)
	CompleteTask(ctx context.Context, req taskcontract.CompleteTaskRequest) (taskcontract.TaskResponse, error)
}
