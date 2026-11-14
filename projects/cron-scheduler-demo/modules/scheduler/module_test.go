package scheduler_test

import (
	"context"
	"testing"

	"github.com/example/cron-scheduler-demo/modules/scheduler"
	"github.com/example/cron-scheduler-demo/modules/scheduler/contract"
	"github.com/go-monolith/mono/monotest"
)

// TestSchedulerModuleConformance checks the scheduler's boundaries without
// NATS. monotest registers the cron services but does not fire them (cron needs
// JetStream; see main_test.go for the real-app test), so the exercise only
// calls the published GetStats contract.
func TestSchedulerModuleConformance(t *testing.T) {
	monotest.CheckModule(t, scheduler.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			resp, err := contract.GetStats.Call(ctx, h.Container(), contract.GetStatsRequest{})
			if err != nil {
				return err
			}
			if len(resp.Schedules) != 0 {
				t.Errorf("no cron fired under monotest, got %+v", resp.Schedules)
			}
			return nil
		}))
}
