package api

import (
	"context"

	schedulercontract "github.com/example/cron-scheduler-demo/modules/scheduler/contract"
)

// StatsPort is the driven port the HTTP handlers need: the cron activity, in
// the scheduler module's contract DTOs. The api module owns it; it says
// nothing about NATS.
type StatsPort interface {
	GetStats(ctx context.Context) (map[string]schedulercontract.ScheduleStat, error)
}
