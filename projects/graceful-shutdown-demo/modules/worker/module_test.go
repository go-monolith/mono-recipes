package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/example/graceful-shutdown-demo/modules/worker"
	"github.com/go-monolith/mono/monotest"
)

// TestWorkerConformance checks the module with monotest. The worker runs a
// task every 20ms that would take 10s, so a task is in progress when Stop is
// called: Stop must interrupt it and return within the stop timeout (rule 8).
func TestWorkerConformance(t *testing.T) {
	monotest.CheckModule(t, worker.NewModule(worker.Config{Interval: 20 * time.Millisecond, TaskDuration: 10 * time.Second}),
		monotest.WithStopTimeout(time.Second),
		monotest.WithExercise(func(context.Context, *monotest.Harness) error {
			time.Sleep(100 * time.Millisecond) // a task is now in progress
			return nil
		}))
}

// TestWorkerStopWithoutStart checks that Stop on a never-started worker is a no-op.
func TestWorkerStopWithoutStart(t *testing.T) {
	if err := worker.NewModule(worker.Config{}).Stop(context.Background()); err != nil {
		t.Fatalf("Stop without Start: %v", err)
	}
}
