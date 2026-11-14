package httpserver_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/graceful-shutdown-demo/modules/httpserver"
	"github.com/go-monolith/mono/monotest"
)

// client opens a fresh connection per request and closes it afterwards. With
// pooled keep-alive connections the transport may dial a spare connection that
// never sends a request; fasthttp's shutdown waits up to 5s for such a "new"
// connection, which would make Stop timings flaky.
var client = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 30 * time.Second}

func get(ctx context.Context, m *httpserver.HttpServerModule, path string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+m.Addr()+path, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

// TestHTTPServerConformance checks the module with monotest (rule 8 is the
// point here: Stop must return within the stop timeout). The exercise leaves a
// /slow request in flight, so Stop has to drain it before returning; the
// request must still complete with 200.
func TestHTTPServerConformance(t *testing.T) {
	m := httpserver.NewModule(httpserver.Config{Addr: "127.0.0.1:0", SlowDelay: 300 * time.Millisecond})
	slow := make(chan error, 1)
	monotest.CheckModule(t, m,
		monotest.WithStopTimeout(2*time.Second),
		monotest.WithExercise(func(ctx context.Context, _ *monotest.Harness) error {
			for _, path := range []string{"/", "/health"} {
				status, err := get(ctx, m, path)
				if err != nil {
					return err
				}
				if status != http.StatusOK {
					return fmt.Errorf("GET %s: status %d", path, status)
				}
			}
			go func() {
				status, err := get(context.Background(), m, "/slow")
				if err == nil && status != http.StatusOK {
					err = fmt.Errorf("GET /slow: status %d", status)
				}
				slow <- err
			}()
			time.Sleep(100 * time.Millisecond) // let /slow reach the handler
			return nil
		}))
	select {
	case err := <-slow:
		if err != nil {
			t.Errorf("in-flight /slow request was not drained: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("in-flight /slow request did not finish")
	}
}

// TestHTTPServerStopHonoursDeadline checks that Stop returns at the shutdown
// deadline even when an in-flight request would take longer, and that the
// request is cancelled rather than left running.
func TestHTTPServerStopHonoursDeadline(t *testing.T) {
	m := httpserver.NewModule(httpserver.Config{Addr: "127.0.0.1:0", SlowDelay: 10 * time.Second})
	h := monotest.New(t, m, monotest.Strict())
	if err := h.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	slowDone := make(chan time.Time, 1)
	go func() {
		_, _ = get(context.Background(), m, "/slow")
		slowDone <- time.Now()
	}()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := h.Stop(ctx)
	elapsed := time.Since(start)
	// The harness wraps the module's error as text, so match the message.
	if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Errorf("Stop error = %v, want a deadline exceeded error", err)
	}
	if elapsed > time.Second {
		t.Errorf("Stop took %s, want it bounded by the 300ms deadline", elapsed)
	}
	select {
	case done := <-slowDone:
		if d := done.Sub(start); d > 2*time.Second {
			t.Errorf("in-flight request ended %s after Stop, want it cancelled at the deadline", d)
		}
	case <-time.After(3 * time.Second):
		t.Error("in-flight request was not cancelled at the shutdown deadline")
	}
}
