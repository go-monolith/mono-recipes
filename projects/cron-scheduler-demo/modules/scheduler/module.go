package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/example/cron-scheduler-demo/stats"
	"github.com/go-monolith/mono"
)

// Module registers a set of server-side cron-scheduled services that showcase
// the mono CronService type. Each schedule is provisioned in the NATS JetStream
// message scheduler, so in a multi-node deployment exactly one occurrence fires
// per tick — no client-side ticker and no leader election. Occurrences are
// delivered through a durable pull consumer with explicit acknowledgement
// (at-least-once).
type Module struct {
	store *stats.Store
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates the scheduler module, writing tick counts into the shared store.
func NewModule(store *stats.Store) *Module {
	return &Module{store: store}
}

// Name identifies the module.
func (m *Module) Name() string { return "scheduler" }

// Start is a no-op; the cron schedules are provisioned by the framework when
// RegisterServices runs. The consumers are started automatically.
func (m *Module) Start(_ context.Context) error {
	log.Println("[scheduler] cron services armed: heartbeat (@every 2s), metrics (*/20 * * * * *, 6-field), daily-report (0 0 * * *, 5-field)")
	return nil
}

// Stop is a no-op; the framework drains the cron consumers on shutdown.
func (m *Module) Stop(_ context.Context) error {
	log.Println("[scheduler] stopped; framework drains the cron consumers")
	return nil
}

// RegisterServices registers three cron-scheduled services. Each demonstrates a
// different schedule format accepted by CronServiceConfig.Schedule:
//
//   - "@every 2s"        interval form (minimum 1s; no time zone)
//   - "*/20 * * * * *"   six-field cron expression (here: every 20s)
//   - "0 0 * * *"        standard five-field cron expression (here: daily 00:00)
//
// NOTE on cron syntax: mono v0.0.8+ accepts BOTH the standard five-field cron
// format and the embedded scheduler's native six-field (seconds-first) format.
// A five-field expression is normalized by prepending a "0" seconds field, so
// "0 0 * * *" becomes "0 0 0 * * *" (daily at midnight) — see
// types.NormalizeCronSchedule.
//
//	┌──────────── second (0-59)   ← six-field only; omit for five-field
//	│ ┌────────── minute (0-59)
//	│ │ ┌──────── hour (0-23)
//	│ │ │ ┌────── day of month (1-31)
//	│ │ │ │ ┌──── month (1-12)
//	│ │ │ │ │ ┌── day of week (0-6, Sun=0)
//	* * * * * *
//
// Step (*/n), ranges (1-5) and lists (1,3,5) are supported. Named aliases
// (@daily, @hourly, ...) and intervals (@every 5m) work too.
//
// The framework owns acknowledgement: returning nil Acks the occurrence, a
// non-nil error Naks it (redelivered up to the consumer's MaxDeliver).
func (m *Module) RegisterServices(c mono.ServiceContainer) error {
	// 1) Interval schedule with a static payload — a liveness heartbeat.
	if err := c.RegisterCronService("heartbeat", mono.CronServiceConfig{
		Schedule: "@every 2s",
		Payload:  []byte(`{"event":"heartbeat"}`),
	}, m.onHeartbeat); err != nil {
		return fmt.Errorf("register heartbeat cron: %w", err)
	}

	// 2) Cron expression with an explicit time zone and a per-message TTL.
	//    Fires every 20s (at seconds 0/20/40 of each minute, UTC). The TTL drops
	//    any occurrence not delivered within 15s (e.g. all consumers down),
	//    avoiding a backlog of stale rollups.
	if err := c.RegisterCronService("metrics", mono.CronServiceConfig{
		Schedule: "*/20 * * * * *",
		TimeZone: "UTC",
		TTL:      15 * time.Second,
		Payload:  []byte(`{"event":"metrics-rollup"}`),
	}, m.onMetrics); err != nil {
		return fmt.Errorf("register metrics cron: %w", err)
	}

	// 3) Standard FIVE-field cron expression — a once-a-day summary at 00:00 UTC.
	//    "0 0 * * *" is normalized to "0 0 0 * * *" by the framework (v0.0.8+).
	//    In a real app this is where you'd email a daily digest. (The named
	//    aliases @daily/@hourly/... are accepted here too.)
	if err := c.RegisterCronService("daily-report", mono.CronServiceConfig{
		Schedule: "0 0 * * *",
		TimeZone: "UTC",
		Payload:  []byte(`{"event":"daily-report"}`),
	}, m.onDailyReport); err != nil {
		return fmt.Errorf("register daily-report cron: %w", err)
	}

	// Two more CronServiceConfig options worth knowing (not wired here so the
	// demo stays runnable without an extra producer):
	//
	//   SourceSubject: instead of a static Payload, deliver the *last* message
	//     seen on a subject on each tick — handy for downsampling a busy stream.
	//     Mutually exclusive with Payload.
	//
	//   Deprecated: set true in a release to purge the server-side schedule and
	//     stop the consumer while keeping this registration in place (phase one
	//     of a safe two-phase retirement), then delete the call in a later one.
	return nil
}

func (m *Module) onHeartbeat(_ context.Context, msg *mono.Msg) error {
	n := m.store.Record("heartbeat")
	log.Printf("[scheduler] heartbeat #%d: %s", n, msg.Data)
	return nil
}

func (m *Module) onMetrics(_ context.Context, _ *mono.Msg) error {
	n := m.store.Record("metrics")
	snap := m.store.Snapshot()
	log.Printf("[scheduler] metrics rollup #%d at %s — heartbeats so far: %d",
		n, time.Now().UTC().Format(time.RFC3339), snap["heartbeat"].Ticks)
	return nil
}

func (m *Module) onDailyReport(_ context.Context, msg *mono.Msg) error {
	n := m.store.Record("daily-report")
	log.Printf("[scheduler] daily report #%d: %s", n, msg.Data)
	return nil
}
