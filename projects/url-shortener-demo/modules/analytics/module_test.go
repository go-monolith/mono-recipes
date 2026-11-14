package analytics_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/url-shortener-demo/modules/analytics"
	"github.com/example/url-shortener-demo/modules/analytics/contract"
	shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"
)

// nopLogger is the module logger in tests (the harness logs framework events).
type nopLogger struct{}

func (nopLogger) Debug(string, ...any)             {}
func (nopLogger) Info(string, ...any)              {}
func (nopLogger) Warn(string, ...any)              {}
func (nopLogger) Error(string, ...any)             {}
func (l nopLogger) With(...any) types.Logger       { return l }
func (l nopLogger) WithModule(string) types.Logger { return l }
func (l nopLogger) WithError(error) types.Logger   { return l }

// publishShortenerEvents plays the shortener: it publishes one URLCreated and
// one URLAccessed event on the harness bus through the shortener's contract.
func publishShortenerEvents(h *monotest.Harness) error {
	now := time.Now()
	if err := shortenercontract.URLCreatedV1.Publish(h.EventBus(), shortenercontract.URLCreatedEvent{
		ShortCode: "abc123", OriginalURL: "https://example.com", CreatedAt: now,
	}, nil); err != nil {
		return err
	}
	return shortenercontract.URLAccessedV1.Publish(h.EventBus(), shortenercontract.URLAccessedEvent{
		ShortCode: "abc123", OriginalURL: "https://example.com", AccessedAt: now, UserAgent: "test",
	}, nil)
}

// waitForSummary polls get-analytics-summary until the events were consumed.
func waitForSummary(ctx context.Context, h *monotest.Harness) (contract.Summary, error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		summary, err := contract.GetSummary.Call(ctx, h.Container(), contract.SummaryRequest{})
		if err != nil {
			return summary, err
		}
		if summary.URLsCreated == 1 && summary.TotalAccesses == 1 {
			return summary, nil
		}
		if time.Now().After(deadline) {
			return summary, fmt.Errorf("events not consumed, summary %+v", summary)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestAnalyticsModuleConformance is the module's boundary check: it consumes
// the shortener events through the contract definitions and every provided
// contract is called.
func TestAnalyticsModuleConformance(t *testing.T) {
	monotest.CheckModule(t, analytics.NewModule(nopLogger{}),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			if err := publishShortenerEvents(h); err != nil {
				return err
			}
			if _, err := waitForSummary(ctx, h); err != nil {
				return err
			}
			logs, err := contract.GetLogs.Call(ctx, h.Container(), contract.LogsRequest{Limit: 10})
			if err != nil {
				return err
			}
			if len(logs) != 1 || logs[0].ShortCode != "abc123" {
				return fmt.Errorf("unexpected logs %+v", logs)
			}
			return nil
		}))
}
