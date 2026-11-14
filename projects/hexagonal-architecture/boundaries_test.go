package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/hexagonal-architecture/modules/api"
	"github.com/example/hexagonal-architecture/modules/notification"
	"github.com/example/hexagonal-architecture/modules/task"
	"github.com/example/hexagonal-architecture/modules/user"
)

// TestModuleBoundaries runs the whole application on an in-process NATS server
// in the default (recording) mode, creates a task through the HTTP API (api ->
// task -> user, then a TaskCreated event to notification), waits for the
// notification, and asserts the boundary report is clean: no violations and no declared
// dependency left unused. This is what makes WithStrictModuleBoundaries() in
// main.go safe to keep on.
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
	apiModule := api.NewModuleWithAddr("127.0.0.1:0")
	notificationModule := notification.NewModule()
	for _, m := range []mono.Module{user.NewModule(), notificationModule, task.NewModule(), apiModule} {
		if err := app.Register(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.Stop(ctx) }()

	req := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader(`{"title":"Buy milk","user_id":"user-1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := apiModule.App().Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create task: status %d", resp.StatusCode)
	}

	deadline := time.Now().Add(5 * time.Second)
	for len(notificationModule.GetNotifications()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("notification module never received the TaskCreated event")
		}
		time.Sleep(20 * time.Millisecond)
	}

	reports := app.ModuleBoundaries()
	if len(reports) != 4 {
		t.Fatalf("got %d boundary reports, want 4", len(reports))
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
