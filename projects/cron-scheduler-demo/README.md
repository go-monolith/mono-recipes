# Cron Scheduler with CronService

This recipe demonstrates the mono framework's `CronService` — a **server-side**,
cron-scheduled service backed by the embedded NATS JetStream **message scheduler**
(nats-server v2.14+). You register a schedule and a handler; the framework provisions
the schedule on the server and invokes your handler on every occurrence.

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
                          │  MONO_CRON_scheduler_daily-report @daily   │
                          └───────────────┬───────────────────────────┘
                                          │ one delivery per occurrence
                                          ▼
                          scheduler handlers  ──record──►  stats.Store  ◄──read── api module
                          (Ack on nil / Nak on error)                            GET /stats
```

- **`modules/scheduler`** — a `ServiceProviderModule` that registers three cron services
  in `RegisterServices`, each handler recording a tick in a shared `stats.Store`.
- **`modules/api`** — a Fiber HTTP module exposing `GET /health` and `GET /stats` so you
  can observe cron activity live.
- **`stats`** — a tiny thread-safe counter store shared by both modules.

## The three schedules (schedule-format showcase)

| Service        | `Schedule`         | Form              | Notes                                   |
| -------------- | ------------------ | ----------------- | --------------------------------------- |
| `heartbeat`    | `@every 2s`        | interval          | minimum `1s`; no time zone              |
| `metrics`      | `*/20 * * * * *`   | cron expression   | every 20s; `TimeZone: "UTC"`, `TTL: 15s`|
| `daily-report` | `@daily`           | named alias       | 00:00 UTC (also `@hourly`, `@weekly`, …)|

### ⚠️ Cron expressions are SIX fields (seconds first)

The embedded scheduler uses a six-field cron format — a leading **seconds** field:

```
┌──────────── second (0-59)
│ ┌────────── minute (0-59)
│ │ ┌──────── hour (0-23)
│ │ │ ┌────── day of month (1-31)
│ │ │ │ ┌──── month (1-12)
│ │ │ │ │ ┌── day of week (0-6, Sunday = 0)
* * * * * *
```

Steps (`*/n`), ranges (`1-5`) and lists (`1,3,5`) are supported. A familiar five-field
expression such as `0 0 * * *` is **rejected** — write `0 0 0 * * *` instead. Named aliases
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

## Project layout

```
cron-scheduler-demo/
├── main.go                  # JetStream-enabled app, module wiring, graceful shutdown
├── stats/store.go           # thread-safe tick counters shared by both modules
└── modules/
    ├── scheduler/module.go  # registers the three CronServices (the showcase)
    └── api/module.go         # Fiber HTTP server: GET /health, GET /stats
```
