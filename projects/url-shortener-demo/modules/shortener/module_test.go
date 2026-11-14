package shortener_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/mono"
	monoerrors "github.com/go-monolith/mono/pkg/errors"
	"github.com/go-monolith/mono/pkg/helper"
	kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"

	"github.com/example/url-shortener-demo/modules/shortener"
	"github.com/example/url-shortener-demo/modules/shortener/contract"
)

// The shortener stores URLs in the kv-jetstream plugin, which needs the
// embedded NATS JetStream server, so monotest (NATS-free) cannot run it.
// This is the module's boundary test instead: a real application in strict
// mode, with the plugin, the shortener and a probe module that consumes the
// shortener's events through the contract definitions.

// eventProbe is a stand-in consumer of the shortener events.
type eventProbe struct {
	mu       sync.Mutex
	created  []contract.URLCreatedEvent
	accessed []contract.URLAccessedEvent
}

var _ mono.EventConsumerModule = (*eventProbe)(nil)

func (p *eventProbe) Name() string                  { return "event-probe" }
func (p *eventProbe) Start(_ context.Context) error { return nil }
func (p *eventProbe) Stop(_ context.Context) error  { return nil }

func (p *eventProbe) RegisterEventConsumers(registry mono.EventRegistry) error {
	if err := helper.RegisterTypedEventConsumer(registry, contract.URLCreatedV1,
		func(_ context.Context, e contract.URLCreatedEvent, _ *mono.Msg) error {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.created = append(p.created, e)
			return nil
		}, p); err != nil {
		return err
	}
	return helper.RegisterTypedEventConsumer(registry, contract.URLAccessedV1,
		func(_ context.Context, e contract.URLAccessedEvent, _ *mono.Msg) error {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.accessed = append(p.accessed, e)
			return nil
		}, p)
}

func (p *eventProbe) counts() (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.created), len(p.accessed)
}

func startApp(t *testing.T) (mono.MonoApplication, *eventProbe) {
	t.Helper()
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithJetStreamStorageDir(t.TempDir()),
		// In-process NATS: no TCP port, so parallel test runs cannot collide.
		mono.WithNATSDontListen(),
		mono.WithNATSInProcessConn(),
		// Default (non-strict) mode on purpose: violations are recorded in
		// app.ModuleBoundaries() instead of rejected, so the report check below
		// is meaningful. main.go runs strict once this report is clean.
	)
	if err != nil {
		t.Fatal(err)
	}
	kv, err := kvjetstream.New(kvjetstream.Config{
		Buckets: []kvjetstream.BucketConfig{{Name: "urls", Storage: kvjetstream.MemoryStorage}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RegisterPlugin(kv, "kv"); err != nil {
		t.Fatal(err)
	}
	probe := &eventProbe{}
	for _, m := range []mono.Module{shortener.NewModule("http://short.test", app.Logger()), probe} {
		if err := app.Register(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	return app, probe
}

// TestShortenerModuleBoundaries calls every contract the shortener provides,
// through the contracts, and checks the events and the boundary report.
func TestShortenerModuleBoundaries(t *testing.T) {
	app, probe := startApp(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svc := app.Services(contract.ModuleName)

	created, err := contract.ShortenURL.Call(ctx, svc, contract.ShortenRequest{URL: "https://example.com/a"})
	if err != nil {
		t.Fatalf("shorten-url: %v", err)
	}
	if created.ShortCode == "" || created.ShortURL != "http://short.test/"+created.ShortCode {
		t.Fatalf("unexpected shorten response: %+v", created)
	}

	resolved, err := contract.ResolveURL.Call(ctx, svc, contract.ResolveRequest{ShortCode: created.ShortCode, UserAgent: "test"})
	if err != nil {
		t.Fatalf("resolve-url: %v", err)
	}
	if resolved.OriginalURL != "https://example.com/a" {
		t.Errorf("resolved %q", resolved.OriginalURL)
	}

	stats, err := contract.GetStats.Call(ctx, svc, contract.ShortCodeRequest{ShortCode: created.ShortCode})
	if err != nil {
		t.Fatalf("get-stats: %v", err)
	}
	if stats.AccessCount != 1 {
		t.Errorf("access count = %d, want 1", stats.AccessCount)
	}

	urls, err := contract.ListURLs.Call(ctx, svc, contract.ListURLsRequest{})
	if err != nil {
		t.Fatalf("list-urls: %v", err)
	}
	if len(urls) != 1 {
		t.Errorf("listed %d urls, want 1", len(urls))
	}

	deleted, err := contract.DeleteURL.Call(ctx, svc, contract.ShortCodeRequest{ShortCode: created.ShortCode})
	if err != nil {
		t.Fatalf("delete-url: %v", err)
	}
	if deleted.ShortCode != created.ShortCode {
		t.Errorf("deleted %q", deleted.ShortCode)
	}

	// Handler errors come back as RemoteErrors carrying the contract error.
	_, err = contract.GetStats.Call(ctx, svc, contract.ShortCodeRequest{ShortCode: created.ShortCode})
	assertRemoteError(t, err, contract.ErrURLNotFound)
	_, err = contract.ShortenURL.Call(ctx, svc, contract.ShortenRequest{URL: "ftp://example.com"})
	assertRemoteError(t, err, contract.ErrInvalidURL)
	_, err = contract.ResolveURL.Call(ctx, svc, contract.ResolveRequest{ShortCode: "bad-code"})
	assertRemoteError(t, err, contract.ErrInvalidShortCode)

	// Both events reached the probe through the contract definitions.
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, a := probe.counts()
		if c == 1 && a == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("events received: created=%d accessed=%d, want 1 and 1", c, a)
		}
		time.Sleep(20 * time.Millisecond)
	}

	for _, r := range app.ModuleBoundaries() {
		if len(r.Violations) != 0 {
			t.Errorf("module %s boundary violations: %+v", r.Module, r.Violations)
		}
	}
}

func assertRemoteError(t *testing.T, err error, want error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %v, got nil", want)
	}
	remote, ok := monoerrors.GetRemoteError(err)
	if !ok {
		t.Fatalf("expected a RemoteError, got %v", err)
	}
	if !strings.HasPrefix(remote.Message, want.Error()) {
		t.Errorf("remote message %q does not start with %q", remote.Message, want.Error())
	}
	if errors.Is(err, want) {
		t.Errorf("a RemoteError should not unwrap to the sentinel (only its message travels)")
	}
}
