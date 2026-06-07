// cron-service.go demonstrates CronService — server-side scheduled services
// backed by the embedded NATS JetStream message scheduler (nats-server v2.14+).
//
// The schedule is registered server-side, so in a multi-node deployment exactly
// one occurrence fires per tick — no client-side ticker and no leader election.
// Each occurrence is delivered through a durable pull consumer with explicit
// acknowledgement (at-least-once): return nil to Ack, or a non-nil error to Nak.
//
// CronService requires JetStream, so the app must be created with a storage dir:
//
//	app, _ := mono.NewMonoApplication(
//	    mono.WithJetStreamStorageDir(filepath.Join(os.TempDir(), "my-app")),
//	)
package maintenance

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono"
)

// Module provides scheduled maintenance jobs via CronService.
type Module struct {
	logger mono.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new maintenance module
func NewModule(logger mono.Logger) *Module {
	return &Module{logger: logger}
}

// Name returns the module name
func (m *Module) Name() string { return "maintenance" }

// RegisterServices registers cron-scheduled services. Each demonstrates one of
// the schedule formats accepted by CronServiceConfig.Schedule:
//
//   - "@every 30s"   interval (minimum 1s; no time zone)
//   - "0 */5 * * * *" six-field cron, seconds-first (here: every 5 minutes)
//   - "0 3 * * *"    standard five-field cron (here: daily 03:00); normalized
//     to "0 0 3 * * *" by prepending a "0" seconds field
//   - "@daily"       named alias (also @hourly, @weekly, @monthly, @yearly)
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// 1) Interval with a static payload — a frequent heartbeat/health probe.
	if err := container.RegisterCronService(
		"heartbeat",
		mono.CronServiceConfig{
			Schedule: "@every 30s",
			Payload:  []byte(`{"event":"heartbeat"}`),
		},
		m.handleHeartbeat,
	); err != nil {
		return fmt.Errorf("failed to register heartbeat cron: %w", err)
	}

	// 2) Five-field cron with an explicit time zone and a per-occurrence TTL.
	//    The TTL drops an occurrence that isn't delivered in time (e.g. all
	//    consumers were down), preventing a backlog of stale work.
	if err := container.RegisterCronService(
		"cleanup",
		mono.CronServiceConfig{
			Schedule: "0 3 * * *", // daily at 03:00 (normalized to "0 0 3 * * *")
			TimeZone: "UTC",
			TTL:      1 * time.Hour,
			Payload:  []byte(`{"event":"nightly-cleanup"}`),
		},
		m.handleCleanup,
	); err != nil {
		return fmt.Errorf("failed to register cleanup cron: %w", err)
	}

	log.Println("[maintenance] Registered cron services: heartbeat (@every 30s), cleanup (0 3 * * *)")
	return nil
}

// Start initializes the maintenance module
func (m *Module) Start(_ context.Context) error {
	log.Println("[maintenance] Module started")
	return nil
}

// Stop cleans up the maintenance module; the framework drains the cron consumers.
func (m *Module) Stop(_ context.Context) error {
	log.Println("[maintenance] Module stopped")
	return nil
}

// Handler implementations. The framework owns acknowledgement: returning nil
// Acks the occurrence, a non-nil error Naks it (redelivered up to MaxDeliver).
// msg.Data carries the configured Payload (or the last SourceSubject message).

func (m *Module) handleHeartbeat(_ context.Context, msg *mono.Msg) error {
	m.logger.Info("heartbeat tick", "payload", string(msg.Data))
	return nil
}

func (m *Module) handleCleanup(ctx context.Context, _ *mono.Msg) error {
	removed, err := m.purgeExpiredRecords(ctx)
	if err != nil {
		// Returning the error Naks the occurrence; the server will redeliver it.
		m.logger.Error("nightly cleanup failed", "error", err)
		return err
	}
	m.logger.Info("nightly cleanup complete", "records_removed", removed)
	return nil
}

func (m *Module) purgeExpiredRecords(_ context.Context) (int, error) {
	// ... delete expired rows / files / cache entries ...
	return 0, nil
}

// Retiring a schedule safely (two-phase): removing the RegisterCronService call
// does NOT delete the server-side schedule by itself. First ship a release with
// Deprecated:true (purges the schedule and stops the consumer while keeping the
// registration), then remove the call in a later release.
//
//	container.RegisterCronService("cleanup", mono.CronServiceConfig{
//	    Schedule:   "0 3 * * *",
//	    Deprecated: true, // phase one: purge + stop, keep code in place
//	}, m.handleCleanup)
//
// To downsample a busy subject instead of a static Payload, set SourceSubject to
// that subject (mutually exclusive with Payload); the last message seen on it is
// delivered on each tick.
