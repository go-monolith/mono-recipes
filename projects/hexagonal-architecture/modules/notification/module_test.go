package notification_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"

	"github.com/example/hexagonal-architecture/modules/notification"
	taskcontract "github.com/example/hexagonal-architecture/modules/task/contract"
)

// TestNotificationModuleConformance checks the notification module against
// the Ports and Adapters rules. The exercise publishes each task event through
// the task contract's definitions (as the task module would) and waits for the
// consumers to log a notification for each.
func TestNotificationModuleConformance(t *testing.T) {
	module := notification.NewModule()
	monotest.CheckModule(t, module,
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			bus := h.EventBus()
			now := time.Now()
			if err := taskcontract.TaskCreatedV1.Publish(bus, taskcontract.TaskCreatedEvent{TaskID: "t-1", Title: "Buy milk", UserID: "user-1", CreatedAt: now}, nil); err != nil {
				return err
			}
			if err := taskcontract.TaskCompletedV1.Publish(bus, taskcontract.TaskCompletedEvent{TaskID: "t-1", UserID: "user-1", CompletedAt: now}, nil); err != nil {
				return err
			}
			if err := taskcontract.TaskDeletedV1.Publish(bus, taskcontract.TaskDeletedEvent{TaskID: "t-1", UserID: "user-1", DeletedAt: now}, nil); err != nil {
				return err
			}
			return waitFor(ctx, func() bool { return len(module.GetNotifications()) == 3 })
		}))

	want := map[string]bool{"task_created": true, "task_completed": true, "task_deleted": true}
	for _, n := range module.GetNotifications() {
		delete(want, n.Type)
	}
	if len(want) != 0 {
		t.Errorf("missing notifications: %v", want)
	}
}

func waitFor(ctx context.Context, cond func() bool) error {
	for !cond() {
		select {
		case <-ctx.Done():
			return fmt.Errorf("condition not met: %w", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
	return nil
}
