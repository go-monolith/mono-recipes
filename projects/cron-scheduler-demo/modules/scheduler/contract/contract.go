// Package contract is the published port of the scheduler module: the DTOs of
// its services and the typed contracts other modules call them through. It is
// the ONLY package of the scheduler module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper. Never its own
// module package, never another module's non-contract package.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// ModuleName is the name of the module that provides these contracts.
const ModuleName = "scheduler"

// ScheduleStat is the public view of a single schedule's activity. LastTick is a
// pointer so it is genuinely omitted from JSON (via omitempty) for a schedule
// that has not fired yet — a struct time.Time would always be serialized.
type ScheduleStat struct {
	Ticks    int64      `json:"ticks"`
	LastTick *time.Time `json:"last_tick,omitempty"`
}

// GetStatsRequest asks for the activity of every cron schedule. It has no fields.
type GetStatsRequest struct{}

// GetStatsResponse holds per-schedule tick counts keyed by schedule name
// (heartbeat, metrics, daily-report). A schedule that has not fired yet is absent.
type GetStatsResponse struct {
	Schedules map[string]ScheduleStat `json:"schedules"`
}

// GetStats is the request-reply service (services.scheduler.get-stats) that
// returns a snapshot of the cron activity recorded by the scheduler module.
var GetStats = helper.RequestReplyService[GetStatsRequest, GetStatsResponse](ModuleName, "get-stats")
