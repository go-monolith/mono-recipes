# Cron Scheduler with CronService

This recipe demonstrates the mono framework's `CronService` — a **server-side**,
cron-scheduled service backed by the embedded NATS JetStream **message scheduler**
(nats-server v2.14+). You register a schedule and a handler; the framework provisions
the schedule on the server and invokes your handler on every occurrence.

Built on **mono v0.1.1**. The two modules talk only through a typed service contract
(`scheduler/contract.GetStats`), and the app runs with strict module boundaries.

> Reference: [Mono docs — Cron Scheduler](https://gelmium.gitbook.io/monolith-framework/examples/cron-scheduler)

## Why CronService instead of a `time.Ticker`?

A naive in-process ticker runs on **every** node. In a multi-node deployment that
means N duplicate firings per tick, so you end up hand-rolling leader election. With
`CronService` the schedule lives in JetStream, so:

- **Exactly one occurrence fires per tick across the whole cluster** — no client-side
  ticker, no leader election.
- **At-least-once delivery with explicit ack** — each occurrence is delivered through a
  durable pull consumer. Return `nil` to **Ack**; return an error to **Nak** (redelivered
  up to the consumer's `MaxDeliver`).
- **Survives restarts** — the schedule is persisted; the server skips ahead so a restart
  doesn't replay a backlog.

## Architecture

```
                          embedded NATS (JetStream message scheduler)
                          ┌───────────────────────────────────────────┐
   scheduler module       │  MONO_CRON_scheduler_heartbeat  @every 2s  │
   registers 3 schedules ─┼─►MONO_CRON_scheduler_metrics    */20 * * * * *
                          │  MONO_CRON_scheduler_daily-report 0 0 * * *│
                          └───────────────┬───────────────────────────┘
                                          │ one delivery per occurrence
                                          ▼
                          scheduler handlers ──record──► stats store (private to scheduler)
                          (Ack on nil / Nak on error)          │
                                                               │ contract.GetStats
                                                               │ services.scheduler.get-stats
                                                               ▼
                                     api module: StatsPort ─► schedulerAdapter ─► GET /stats
```

- **`modules/scheduler`** — a `ServiceProviderModule` that registers three cron services
  in `RegisterServices`, each handler recording a tick in the module's private stats
  store, plus the `get-stats` request-reply service (through its typed contract) that
  returns a snapshot of it.
- **`modules/scheduler/contract`** — the scheduler's published port: the `ScheduleStat`
  DTO and the `GetStats` contract. The only scheduler package other modules import.
- **`modules/api`** — a Fiber HTTP module exposing `GET /health` and `GET /stats` so you
  can observe cron activity live. It declares `scheduler` as a dependency and reads the
  stats through its own `StatsPort`, implemented by an adapter over `GetStats`.

## The three schedules (schedule-format showcase)

| Service        | `Schedule`         | Form                   | Notes                                   |
| -------------- | ------------------ | ---------------------- | --------------------------------------- |
| `heartbeat`    | `@every 2s`        | interval               | minimum `1s`; no time zone              |
| `metrics`      | `*/20 * * * * *`   | six-field cron         | every 20s; `TimeZone: "UTC"`, `TTL: 15s`|
| `daily-report` | `0 0 * * *`        | five-field cron        | 00:00 UTC; normalized to `0 0 0 * * *`  |

### Cron expressions: five OR six fields

The mono scheduler (v0.1.1, used here) accepts both the standard **five-field** cron
format and the embedded scheduler's native **six-field** (seconds-first) format. A five-field
expression is normalized by prepending a `0` seconds field, so `0 0 * * *` becomes
`0 0 0 * * *` (daily at midnight). Use six fields when you need sub-minute precision.

```
┌──────────── second (0-59)   ← six-field only; omit for five-field
│ ┌────────── minute (0-59)
│ │ ┌──────── hour (0-23)
│ │ │ ┌────── day of month (1-31)
│ │ │ │ ┌──── month (1-12)
│ │ │ │ │ ┌── day of week (0-6, Sunday = 0)
* * * * * *
```

Steps (`*/n`), ranges (`1-5`) and lists (`1,3,5`) are supported. Named aliases
(`@hourly`, `@daily`, …) and intervals (`@every 5m`) are also accepted.

## `CronServiceConfig` options

```go
c.RegisterCronService("heartbeat", mono.CronServiceConfig{
    Schedule: "@every 2s",                 // required
    Payload:  []byte(`{"event":"beat"}`),  // static body delivered each tick
    // SourceSubject: "metrics.raw",        // deliver last msg on a subject instead
    //                                       //   of Payload (downsampling); exclusive
    //                                       //   with Payload
    // TimeZone:   "America/New_York",       // IANA tz for cron eval (default UTC)
    // TTL:        15 * time.Second,         // per-occurrence TTL (>= 1s)
    // Deprecated: true,                     // purge schedule + stop consumer, keep
    //                                       //   registration (safe retirement, phase 1)
}, handler)
```

`CronService` **requires JetStream**, so the app must be created with a storage dir:

```go
app, _ := mono.NewMonoApplication(
    mono.WithJetStreamStorageDir(filepath.Join(os.TempDir(), "mono-cron-scheduler-demo")),
)
```

### Retiring a schedule safely

Removing a `RegisterCronService` call does not by itself delete the server-side schedule.
Retire in two phases: first ship with `Deprecated: true` (purges the schedule and stops the
consumer while keeping the code), then delete the registration in a later release.

## Run it

```bash
# Build (single binary in ./bin, per the repo convention)
go build -o ./bin/cron-scheduler-demo .

# Run (HTTP on :8080 by default; override with HTTP_ADDR)
./bin/cron-scheduler-demo
```

You'll see a heartbeat every 2 seconds:

```
[scheduler] heartbeat #1: {"event":"heartbeat"}
[scheduler] heartbeat #2: {"event":"heartbeat"}
[scheduler] metrics rollup #1 at 2026-... — heartbeats so far: 9
```

Observe the live counters:

```bash
curl -s http://localhost:8080/stats
# {"heartbeat":{"ticks":11,"last_tick":"..."},"metrics":{"ticks":1,"last_tick":"..."}}

curl -s http://localhost:8080/health
# {"status":"healthy"}
```

Press `Ctrl+C` for a graceful shutdown — the framework drains the cron consumers.

## Module boundaries

The api module does not share state with the scheduler. The scheduler publishes a typed
contract, the only scheduler package other modules import:

```go
// modules/scheduler/contract/contract.go
var GetStats = helper.RequestReplyService[GetStatsRequest, GetStatsResponse]("scheduler", "get-stats")

// modules/scheduler/module.go (provider)
if err := contract.GetStats.Register(c, m.getStats); err != nil { ... }
```

The api module declares the dependency, wires its adapter from the read-only container
it receives, and lists the contract it calls in `ConsumeServices()`, which the framework
validates at startup:

```go
func (m *Module) Dependencies() []string { return []string{"scheduler"} }

func (m *Module) SetDependencyServiceContainer(dep string, c mono.ServiceContainer) {
    if dep == "scheduler" {
        m.stats = schedulerAdapter{container: c} // calls schedulercontract.GetStats.Call
    }
}

func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
    return []types.BaseServiceDefinition{schedulercontract.GetStats.ToBase()}
}
```

`main.go` enables `mono.WithStrictModuleBoundaries()`, so a boundary violation fails
instead of being logged at `Stop`.

### Tests

```bash
go test ./...                       # all tests
go test ./modules/... -run Conformance -v
go test -run TestAppCronStatsEndToEnd -v .
```

- `modules/api/module_test.go` and `modules/scheduler/module_test.go` run
  `monotest.CheckModule` (no NATS). monotest registers but does not fire cron services,
  so the scheduler check exercises `GetStats` only.
- `main_test.go` is the real-app test (cron needs JetStream): embedded NATS on a free
  port, JetStream in `t.TempDir()`, strict mode; it waits for a real heartbeat, reads it
  through the contract and `GET /stats`, and asserts `app.ModuleBoundaries()` reports no
  violations and no unused dependencies. Skipped with `-short`.

## Project layout

```
cron-scheduler-demo/
├── main.go                       # JetStream-enabled app, strict boundaries, graceful shutdown
├── main_test.go                  # real-app test: cron fires, /stats, boundary report
└── modules/
    ├── scheduler/
    │   ├── contract/contract.go  # published port: ScheduleStat + GetStats contract
    │   ├── module.go             # registers the three CronServices (the showcase) + get-stats
    │   ├── stats.go              # private thread-safe tick counters
    │   └── module_test.go        # monotest.CheckModule
    └── api/
        ├── module.go             # Fiber HTTP server: GET /health, GET /stats; ConsumeServices
        ├── ports.go              # StatsPort (consumer-owned)
        ├── adapter.go            # StatsPort over schedulercontract.GetStats
        └── module_test.go        # monotest.CheckModule with a stand-in scheduler
```
