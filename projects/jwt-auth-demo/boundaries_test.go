package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-monolith/mono"

	"github.com/example/jwt-auth-demo/modules/api"
	"github.com/example/jwt-auth-demo/modules/auth"
)

// TestModuleBoundaries runs the whole application on an in-process NATS server
// in the default (recording) mode, drives register, login and a protected
// request through the API, and asserts the boundary report is clean: no
// violations and no declared dependency left unused. This is what makes
// WithStrictModuleBoundaries() in main.go safe to keep on.
func TestModuleBoundaries(t *testing.T) {
	ctx := context.Background()
	t.Setenv("JWT_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithNATSDontListen(),
		mono.WithNATSInProcessConn(),
	)
	if err != nil {
		t.Fatal(err)
	}
	apiModule := api.NewModuleWithAddr("127.0.0.1:0")
	for _, m := range []mono.Module{auth.NewModule(), apiModule} {
		if err := app.Register(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.Stop(ctx) }()

	send := func(method, path, bearer, body string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := apiModule.App().Test(req, 10000)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, want, data)
		}
		return data
	}
	creds := `{"email":"alice@example.com","password":"password123"}`
	send("POST", "/api/v1/auth/register", "", creds, 201)
	var tokens api.TokenResponse
	if err := json.Unmarshal(send("POST", "/api/v1/auth/login", "", creds, 200), &tokens); err != nil {
		t.Fatal(err)
	}
	send("GET", "/api/v1/profile", tokens.AccessToken, "", 200)

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
