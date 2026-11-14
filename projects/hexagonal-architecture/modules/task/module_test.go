package task_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/hexagonal-architecture/modules/task"
	"github.com/example/hexagonal-architecture/modules/task/contract"
	usercontract "github.com/example/hexagonal-architecture/modules/user/contract"
)

// standInUser is a NATS-free stand-in for the user module. Its service is
// registered through the user module's own contract; only "user-1" exists.
func standInUser(t *testing.T, bus *monotest.EventBus) *monotest.Container {
	t.Helper()
	c := monotest.NewContainer("user", bus)
	err := usercontract.ValidateUser.Register(c,
		func(_ context.Context, req usercontract.ValidateUserRequest, _ *types.Msg) (usercontract.ValidateUserResponse, error) {
			return usercontract.ValidateUserResponse{Valid: req.UserID == "user-1"}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestTaskModuleConformance checks the task module against the Ports and
// Adapters rules. The exercise calls every contract the module provides, which
// also drives the consumed user contract and publishes all three events.
func TestTaskModuleConformance(t *testing.T) {
	bus := monotest.NewEventBus()
	monotest.CheckModule(t, task.NewModule(),
		monotest.WithDependency("user", standInUser(t, bus)),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			c := h.Container()
			created, err := contract.CreateTask.Call(ctx, c, contract.CreateTaskRequest{Title: "Buy milk", UserID: "user-1"})
			if err != nil {
				return err
			}
			if _, err := contract.GetTask.Call(ctx, c, contract.GetTaskRequest{TaskID: created.ID}); err != nil {
				return err
			}
			if _, err := contract.UpdateTask.Call(ctx, c, contract.UpdateTaskRequest{TaskID: created.ID, Title: "Buy oat milk"}); err != nil {
				return err
			}
			list, err := contract.ListTasks.Call(ctx, c, contract.ListTasksRequest{UserID: "user-1"})
			if err != nil {
				return err
			}
			if list.Total != 1 {
				return fmt.Errorf("list-tasks total = %d, want 1", list.Total)
			}
			done, err := contract.CompleteTask.Call(ctx, c, contract.CompleteTaskRequest{TaskID: created.ID})
			if err != nil {
				return err
			}
			if done.Status != "completed" || done.CompletedAt == nil {
				return fmt.Errorf("complete-task = %+v", done)
			}
			deleted, err := contract.DeleteTask.Call(ctx, c, contract.DeleteTaskRequest{TaskID: created.ID})
			if err != nil {
				return err
			}
			if !deleted.Deleted {
				return fmt.Errorf("delete-task = %+v", deleted)
			}

			published := h.ModulePublished()
			for _, def := range []types.BaseEventDefinition{
				contract.TaskCreatedV1.ToBase(), contract.TaskCompletedV1.ToBase(), contract.TaskDeletedV1.ToBase(),
			} {
				if !slices.Contains(published, def.Subject) {
					return fmt.Errorf("event %s not published (got %v)", def.Subject, published)
				}
			}
			return nil
		}))
}

// TestCreateTaskRejectsUnknownUser checks that the domain consults UserPort.
func TestCreateTaskRejectsUnknownUser(t *testing.T) {
	ctx := context.Background()
	bus := monotest.NewEventBus()
	user := standInUser(t, bus)
	h := monotest.New(t, task.NewModule(), monotest.WithDependency("user", user))
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	if _, err := contract.CreateTask.Call(ctx, h.Container(), contract.CreateTaskRequest{Title: "x", UserID: "user-9"}); err == nil {
		t.Fatal("create-task for unknown user succeeded, want error")
	}
	if n := user.Calls(usercontract.ValidateUser.Name); n != 1 {
		t.Errorf("validate-user called %d times, want 1", n)
	}
	if v := h.Report().Violations; len(v) != 0 {
		t.Errorf("boundary violations: %+v", v)
	}
}
