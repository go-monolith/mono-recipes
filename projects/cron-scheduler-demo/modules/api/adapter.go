package api

import (
	"context"

	schedulercontract "github.com/example/cron-scheduler-demo/modules/scheduler/contract"
	"github.com/go-monolith/mono"
)

// schedulerAdapter is the driven adapter: it implements StatsPort by calling
// the scheduler's GetStats contract with the container received in
// SetDependencyServiceContainer. It is the only place in the api module that
// knows the call crosses a module boundary.
type schedulerAdapter struct{ container mono.ServiceContainer }

var _ StatsPort = schedulerAdapter{}

func (a schedulerAdapter) GetStats(ctx context.Context) (map[string]schedulercontract.ScheduleStat, error) {
	resp, err := schedulercontract.GetStats.Call(ctx, a.container, schedulercontract.GetStatsRequest{})
	if err != nil {
		return nil, err
	}
	if resp.Schedules == nil {
		resp.Schedules = map[string]schedulercontract.ScheduleStat{}
	}
	return resp.Schedules, nil
}
