package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/background-jobs-demo/modules/api"
	"github.com/example/background-jobs-demo/modules/worker"
)

// TestModuleBoundaries runs the whole application on an in-process NATS server
// in the default (recording) mode, creates a job over HTTP and polls it over
// HTTP until the worker finished it, lists jobs, then asserts the boundary report is clean: no
// violations and no declared dependency left unused. This is what makes
// WithStrictModuleBoundaries() in main.go safe to keep on.
func TestModuleBoundaries(t *testing.T) {
	ctx := context.Background()
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithNATSDontListen(),
		mono.WithNATSInProcessConn(),
	)
	if err != nil {
		t.Fatal(err)
	}
	apiModule := api.NewModule(0)
	for _, m := range []mono.Module{worker.NewModule(), apiModule} {
		if err := app.Register(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.Stop(ctx) }()

	send := func(method, path, body string, want int, out any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := apiModule.GetApp().Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, want, data)
		}
		if out != nil {
			if err := json.Unmarshal(data, out); err != nil {
				t.Fatal(err)
			}
		}
	}

	var created api.JobResponse
	send("POST", "/api/v1/jobs", `{"type":"email","payload":{"to":"a@example.com","subject":"Hi","body":"Hello"}}`, 201, &created)

	deadline := time.Now().Add(10 * time.Second)
	for {
		var j api.JobResponse
		send("GET", "/api/v1/jobs/"+created.ID, "", 200, &j)
		if j.Status == "completed" || j.Status == "failed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job %s still %s", j.ID, j.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
	send("GET", "/api/v1/jobs", "", 200, nil)

	reports := app.ModuleBoundaries()
	if len(reports) != 2 {
		t.Fatalf("got %d boundary reports, want 2", len(reports))
	}
	for _, r := range reports {
		if len(r.Violations) != 0 {
			t.Errorf("module %s: boundary violations %+v", r.Module, r.Violations)
		}
		if len(r.UnusedDependencies) != 0 {
			t.Errorf("module %s: unused dependencies %v", r.Module, r.UnusedDependencies)
		}
	}
}
