package api_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/example/cron-scheduler-demo/modules/api"
	schedulercontract "github.com/example/cron-scheduler-demo/modules/scheduler/contract"
	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"
)

var lastTick = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// client opens a fresh connection per request and closes it afterwards. With
// pooled keep-alive connections the transport may dial a spare connection that
// never sends a request; fasthttp's shutdown waits up to 5s for such a "new"
// connection, which would make Stop timings flaky.
var client = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 30 * time.Second}

// standInScheduler is a NATS-free stand-in for the scheduler module, with
// GetStats registered through the scheduler's own contract.
func standInScheduler(t *testing.T) *monotest.Container {
	t.Helper()
	c := monotest.NewContainer(schedulercontract.ModuleName, monotest.NewEventBus())
	err := schedulercontract.GetStats.Register(c,
		func(context.Context, schedulercontract.GetStatsRequest, *types.Msg) (schedulercontract.GetStatsResponse, error) {
			ts := lastTick
			return schedulercontract.GetStatsResponse{Schedules: map[string]schedulercontract.ScheduleStat{
				"heartbeat": {Ticks: 3, LastTick: &ts},
			}}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func get(ctx context.Context, m *api.Module, path string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+m.Addr()+path, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), err
}

// TestAPIModuleConformance checks the api module's boundaries: it declares
// scheduler, consumes only GetStats, and calls it through its dependency
// container when GET /stats is served.
func TestAPIModuleConformance(t *testing.T) {
	m := api.NewModule("127.0.0.1:0")
	monotest.CheckModule(t, m,
		monotest.WithDependency(schedulercontract.ModuleName, standInScheduler(t)),
		monotest.WithExercise(func(ctx context.Context, _ *monotest.Harness) error {
			status, _, err := get(ctx, m, "/stats")
			if err != nil {
				return err
			}
			if status != http.StatusOK {
				return fmt.Errorf("GET /stats: status %d", status)
			}
			return nil
		}))
}

// TestStatsResponseShape checks that /stats keeps its JSON shape: the
// per-schedule map itself, not wrapped in the contract's response envelope.
func TestStatsResponseShape(t *testing.T) {
	ctx := context.Background()
	scheduler := standInScheduler(t)
	m := api.NewModule("127.0.0.1:0")
	h := monotest.New(t, m, monotest.WithDependency(schedulercontract.ModuleName, scheduler), monotest.Strict())
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	status, body, err := get(ctx, m, "/stats")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"heartbeat":{"ticks":3,"last_tick":"2026-01-02T03:04:05Z"}}`
	if status != http.StatusOK || body != want {
		t.Errorf("GET /stats = %d %s, want 200 %s", status, body, want)
	}

	status, body, err = get(ctx, m, "/health")
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || body != `{"status":"healthy"}` {
		t.Errorf("GET /health = %d %s", status, body)
	}

	if n := scheduler.Calls(schedulercontract.GetStats.Name); n != 1 {
		t.Errorf("get-stats called %d times, want 1", n)
	}
}
