package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/hexagonal-architecture/modules/api"
	taskcontract "github.com/example/hexagonal-architecture/modules/task/contract"
)

// standInTask is a NATS-free stand-in for the task module, registered through
// the task module's own contracts. It knows a single task, "t-1".
func standInTask(t *testing.T, bus *monotest.EventBus) *monotest.Container {
	t.Helper()
	c := monotest.NewContainer("task", bus)
	now := time.Now()
	stored := taskcontract.TaskResponse{ID: "t-1", Title: "Buy milk", Status: "pending", UserID: "user-1", CreatedAt: now, UpdatedAt: now}
	notFound := func(id string) error { return fmt.Errorf("task not found: %s", id) }

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(taskcontract.CreateTask.Register(c, func(_ context.Context, req taskcontract.CreateTaskRequest, _ *types.Msg) (taskcontract.CreateTaskResponse, error) {
		return taskcontract.CreateTaskResponse{ID: stored.ID, Title: req.Title, Status: "pending", CreatedAt: now}, nil
	}))
	must(taskcontract.GetTask.Register(c, func(_ context.Context, req taskcontract.GetTaskRequest, _ *types.Msg) (taskcontract.TaskResponse, error) {
		if req.TaskID != stored.ID {
			return taskcontract.TaskResponse{}, notFound(req.TaskID)
		}
		return stored, nil
	}))
	must(taskcontract.UpdateTask.Register(c, func(_ context.Context, req taskcontract.UpdateTaskRequest, _ *types.Msg) (taskcontract.TaskResponse, error) {
		resp := stored
		resp.Title = req.Title
		return resp, nil
	}))
	must(taskcontract.DeleteTask.Register(c, func(_ context.Context, req taskcontract.DeleteTaskRequest, _ *types.Msg) (taskcontract.DeleteTaskResponse, error) {
		return taskcontract.DeleteTaskResponse{Deleted: req.TaskID == stored.ID}, nil
	}))
	must(taskcontract.ListTasks.Register(c, func(context.Context, taskcontract.ListTasksRequest, *types.Msg) (taskcontract.ListTasksResponse, error) {
		return taskcontract.ListTasksResponse{Tasks: []taskcontract.TaskResponse{stored}, Total: 1}, nil
	}))
	must(taskcontract.CompleteTask.Register(c, func(context.Context, taskcontract.CompleteTaskRequest, *types.Msg) (taskcontract.TaskResponse, error) {
		resp := stored
		resp.Status = "completed"
		resp.CompletedAt = &now
		return resp, nil
	}))
	return c
}

// do sends an HTTP request to the module's Fiber app and checks the status.
func do(module *api.APIModule, method, path string, body any, wantStatus int, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := module.App().Test(req, 5000)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		return fmt.Errorf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, wantStatus, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// TestAPIModuleConformance checks the api module against the Ports and
// Adapters rules. The exercise hits every route, so every consumed task
// contract is called through the adapter.
func TestAPIModuleConformance(t *testing.T) {
	bus := monotest.NewEventBus()
	module := api.NewModuleWithAddr("127.0.0.1:0")
	monotest.CheckModule(t, module,
		monotest.WithDependency("task", standInTask(t, bus)),
		monotest.WithExercise(func(context.Context, *monotest.Harness) error {
			var created api.TaskResponse
			if err := do(module, http.MethodPost, "/api/v1/tasks", api.CreateTaskRequest{Title: "Buy milk", UserID: "user-1"}, http.StatusCreated, &created); err != nil {
				return err
			}
			if created.ID != "t-1" {
				return fmt.Errorf("created id = %q", created.ID)
			}
			var list api.ListTasksResponse
			if err := do(module, http.MethodGet, "/api/v1/tasks?user_id=user-1", nil, http.StatusOK, &list); err != nil {
				return err
			}
			if list.Total != 1 {
				return fmt.Errorf("list total = %d", list.Total)
			}
			if err := do(module, http.MethodGet, "/api/v1/tasks/t-1", nil, http.StatusOK, nil); err != nil {
				return err
			}
			if err := do(module, http.MethodPut, "/api/v1/tasks/t-1", api.UpdateTaskRequest{Title: "Buy oat milk"}, http.StatusOK, nil); err != nil {
				return err
			}
			var done api.TaskResponse
			if err := do(module, http.MethodPost, "/api/v1/tasks/t-1/complete", nil, http.StatusOK, &done); err != nil {
				return err
			}
			if done.Status != "completed" {
				return fmt.Errorf("status after complete = %q", done.Status)
			}
			return do(module, http.MethodDelete, "/api/v1/tasks/t-1", nil, http.StatusNoContent, nil)
		}))
}

// TestUnknownTaskIsNotFound maps a contract error to HTTP 404.
func TestUnknownTaskIsNotFound(t *testing.T) {
	ctx := context.Background()
	bus := monotest.NewEventBus()
	module := api.NewModuleWithAddr("127.0.0.1:0")
	h := monotest.New(t, module, monotest.WithDependency("task", standInTask(t, bus)))
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	if err := do(module, http.MethodGet, "/api/v1/tasks/missing", nil, http.StatusNotFound, nil); err != nil {
		t.Error(err)
	}
	if err := do(module, http.MethodDelete, "/api/v1/tasks/missing", nil, http.StatusNotFound, nil); err != nil {
		t.Error(err)
	}
	if err := do(module, http.MethodPost, "/api/v1/tasks", api.CreateTaskRequest{UserID: "user-1"}, http.StatusBadRequest, nil); err != nil {
		t.Error(err)
	}
}
