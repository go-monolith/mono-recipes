package httpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	analyticscontract "github.com/example/url-shortener-demo/modules/analytics/contract"
	"github.com/example/url-shortener-demo/modules/httpserver"
	shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"
)

// nopLogger is the module logger in tests.
type nopLogger struct{}

func (nopLogger) Debug(string, ...any)             {}
func (nopLogger) Info(string, ...any)              {}
func (nopLogger) Warn(string, ...any)              {}
func (nopLogger) Error(string, ...any)             {}
func (l nopLogger) With(...any) types.Logger       { return l }
func (l nopLogger) WithModule(string) types.Logger { return l }
func (l nopLogger) WithError(error) types.Logger   { return l }

// freeAddr returns a localhost address with a free TCP port.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

// standIns returns NATS-free stand-ins for the shortener and analytics
// modules. Their services are registered through the providers' own
// contracts, so a contract change breaks this test at compile time.
func standIns(t *testing.T) (*monotest.Container, *monotest.Container) {
	t.Helper()
	bus := monotest.NewEventBus()
	shortener := monotest.NewContainer(shortenercontract.ModuleName, bus)
	analytics := monotest.NewContainer(analyticscontract.ModuleName, bus)
	now := time.Now()

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(shortenercontract.ShortenURL.Register(shortener,
		func(_ context.Context, req shortenercontract.ShortenRequest, _ *types.Msg) (shortenercontract.ShortenResponse, error) {
			if !strings.HasPrefix(req.URL, "http") {
				return shortenercontract.ShortenResponse{}, fmt.Errorf("%w: scheme must be http or https", shortenercontract.ErrInvalidURL)
			}
			return shortenercontract.ShortenResponse{ShortCode: "abc123", ShortURL: "http://short.test/abc123", OriginalURL: req.URL, CreatedAt: now}, nil
		}))
	must(shortenercontract.ResolveURL.Register(shortener,
		func(_ context.Context, req shortenercontract.ResolveRequest, _ *types.Msg) (shortenercontract.ResolveResponse, error) {
			switch req.ShortCode {
			case "abc123":
				return shortenercontract.ResolveResponse{OriginalURL: "https://example.com"}, nil
			case "expired":
				return shortenercontract.ResolveResponse{}, shortenercontract.ErrURLExpired
			case "bad-code":
				return shortenercontract.ResolveResponse{}, shortenercontract.ErrInvalidShortCode
			}
			return shortenercontract.ResolveResponse{}, shortenercontract.ErrURLNotFound
		}))
	must(shortenercontract.GetStats.Register(shortener,
		func(_ context.Context, req shortenercontract.ShortCodeRequest, _ *types.Msg) (shortenercontract.StatsResponse, error) {
			if req.ShortCode != "abc123" {
				return shortenercontract.StatsResponse{}, shortenercontract.ErrURLNotFound
			}
			return shortenercontract.StatsResponse{ShortCode: "abc123", OriginalURL: "https://example.com", AccessCount: 1}, nil
		}))
	must(shortenercontract.ListURLs.Register(shortener,
		func(context.Context, shortenercontract.ListURLsRequest, *types.Msg) ([]shortenercontract.URLEntry, error) {
			return []shortenercontract.URLEntry{{ShortCode: "abc123", OriginalURL: "https://example.com"}}, nil
		}))
	must(shortenercontract.DeleteURL.Register(shortener,
		func(_ context.Context, req shortenercontract.ShortCodeRequest, _ *types.Msg) (shortenercontract.DeleteResponse, error) {
			return shortenercontract.DeleteResponse{Message: "URL deleted successfully", ShortCode: req.ShortCode}, nil
		}))
	must(analyticscontract.GetSummary.Register(analytics,
		func(context.Context, analyticscontract.SummaryRequest, *types.Msg) (analyticscontract.Summary, error) {
			return analyticscontract.Summary{URLsCreated: 1, URLsTracked: 1, TotalAccesses: 1, AccessLogs: 1}, nil
		}))
	must(analyticscontract.GetLogs.Register(analytics,
		func(_ context.Context, req analyticscontract.LogsRequest, _ *types.Msg) ([]analyticscontract.AccessLog, error) {
			return []analyticscontract.AccessLog{{ShortCode: "abc123", OriginalURL: "https://example.com"}}, nil
		}))
	return shortener, analytics
}

// httpCase is one request of the exercise and the status it must return.
type httpCase struct {
	method, path, body string
	want               int
}

// exerciseCases drives every handler, and therefore every consumed contract,
// including the mapping of each remote contract error (invalid URL, not found,
// expired, invalid short code) back to its HTTP status code.
var exerciseCases = []httpCase{
	{http.MethodGet, "/health", "", http.StatusOK},
	{http.MethodPost, "/api/v1/shorten", `{"url":"https://example.com"}`, http.StatusCreated},
	{http.MethodPost, "/api/v1/shorten", `{"url":"ftp://example.com"}`, http.StatusBadRequest},
	{http.MethodGet, "/abc123", "", http.StatusTemporaryRedirect},
	{http.MethodGet, "/missing", "", http.StatusNotFound},
	{http.MethodGet, "/expired", "", http.StatusGone},
	{http.MethodGet, "/bad-code", "", http.StatusBadRequest},
	{http.MethodGet, "/api/v1/stats/abc123", "", http.StatusOK},
	{http.MethodGet, "/api/v1/stats/missing", "", http.StatusNotFound},
	{http.MethodGet, "/api/v1/urls", "", http.StatusOK},
	{http.MethodDelete, "/api/v1/urls/abc123", "", http.StatusOK},
	{http.MethodGet, "/api/v1/analytics", "", http.StatusOK},
	{http.MethodGet, "/api/v1/analytics/logs?limit=5", "", http.StatusOK},
}

func exercise(addr string) func(ctx context.Context, h *monotest.Harness) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		// Report redirects instead of following them to example.com.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return func(ctx context.Context, _ *monotest.Harness) error {
		for _, tc := range exerciseCases {
			req, err := http.NewRequestWithContext(ctx, tc.method, "http://"+addr+tc.path, strings.NewReader(tc.body))
			if err != nil {
				return err
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				return fmt.Errorf("%s %s: status %d, want %d (body %s)", tc.method, tc.path, resp.StatusCode, tc.want, body)
			}
			if tc.path == "/api/v1/analytics" {
				var summary analyticscontract.Summary
				if err := json.Unmarshal(body, &summary); err != nil || summary.URLsCreated != 1 {
					return fmt.Errorf("analytics body %s: %v", body, err)
				}
			}
		}
		return nil
	}
}

// TestHTTPServerModuleConformance is the module's boundary check: the module
// reaches its dependencies only through their contracts, declares every
// contract in ConsumeServices, and uses both declared dependencies.
func TestHTTPServerModuleConformance(t *testing.T) {
	shortener, analytics := standIns(t)
	addr := freeAddr(t)
	monotest.CheckModule(t, httpserver.NewModule(addr, nopLogger{}),
		monotest.WithDependency(shortenercontract.ModuleName, shortener),
		monotest.WithDependency(analyticscontract.ModuleName, analytics),
		monotest.WithExercise(exercise(addr)))

	for _, svc := range []string{
		shortenercontract.ShortenURL.Name, shortenercontract.ResolveURL.Name, shortenercontract.GetStats.Name,
		shortenercontract.ListURLs.Name, shortenercontract.DeleteURL.Name,
	} {
		if shortener.Calls(svc) == 0 {
			t.Errorf("shortener contract %s was never called", svc)
		}
	}
	for _, svc := range []string{analyticscontract.GetSummary.Name, analyticscontract.GetLogs.Name} {
		if analytics.Calls(svc) == 0 {
			t.Errorf("analytics contract %s was never called", svc)
		}
	}
}
