package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	schedulercontract "github.com/example/cron-scheduler-demo/modules/scheduler/contract"
	"github.com/go-monolith/mono"
)

// The cron services need JetStream, which monotest does not emulate, so this
// test runs the real application as main.go wires it: embedded NATS on a free
// port, JetStream in a temp dir and strict module boundaries.

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func TestAppCronStatsEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for real cron occurrences")
	}
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithNATSPort(freePort(t)),
		mono.WithJetStreamStorageDir(t.TempDir()),
		mono.WithShutdownTimeout(5*time.Second),
		// Default (non-strict) mode on purpose: violations are recorded in
		// app.ModuleBoundaries() instead of rejected, so the report check below
		// is meaningful. main.go runs strict once this report is clean.
	)
	if err != nil {
		t.Fatal(err)
	}
	httpAddr := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	if err := registerModules(app, httpAddr); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			if err := app.Stop(ctx); err != nil {
				t.Errorf("stop: %v", err)
			}
		}
	}
	defer stop()

	// The heartbeat fires every 2s; wait for it through the contract, as the
	// host program may (app.Services is a host-level escape hatch).
	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := schedulercontract.GetStats.Call(ctx, app.Services(schedulercontract.ModuleName), schedulercontract.GetStatsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if hb, ok := resp.Schedules["heartbeat"]; ok && hb.Ticks >= 1 && hb.LastTick != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("heartbeat did not fire within 15s: %+v", resp.Schedules)
		}
		time.Sleep(250 * time.Millisecond)
	}

	// The api module serves the same stats over HTTP via its adapter.
	httpResp, err := http.Get("http://" + httpAddr + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(httpResp.Body)
	_ = httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /stats: %d %s", httpResp.StatusCode, body)
	}
	var stats map[string]schedulercontract.ScheduleStat
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	if stats["heartbeat"].Ticks < 1 {
		t.Errorf("GET /stats = %s, want heartbeat ticks", body)
	}

	apiUsedGetStats := false
	for _, r := range app.ModuleBoundaries() {
		if r.Module == "api" {
			for _, use := range r.Observed {
				if use.Module == schedulercontract.ModuleName && use.Service == schedulercontract.GetStats.Name && use.Lookups > 0 {
					apiUsedGetStats = true
				}
			}
		}
		if len(r.Violations) != 0 {
			t.Errorf("module %s: boundary violations: %+v", r.Module, r.Violations)
		}
		if len(r.UnusedDependencies) != 0 {
			t.Errorf("module %s: unused dependencies: %v", r.Module, r.UnusedDependencies)
		}
	}
	if !apiUsedGetStats {
		t.Errorf("boundary report does not show api looking up scheduler/get-stats: %+v", app.ModuleBoundaries())
	}
	stop()
}
