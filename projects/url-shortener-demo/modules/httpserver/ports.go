package httpserver

import (
	"context"

	analyticscontract "github.com/example/url-shortener-demo/modules/analytics/contract"
	shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"
)

// ShortenerPort is a driven port: what the HTTP handlers need from the
// shortener module. The http-server module owns it, it speaks the shortener's
// contract DTOs, and it says nothing about NATS. Tests can substitute a fake.
type ShortenerPort interface {
	ShortenURL(ctx context.Context, req shortenercontract.ShortenRequest) (shortenercontract.ShortenResponse, error)
	ResolveURL(ctx context.Context, req shortenercontract.ResolveRequest) (string, error)
	GetStats(ctx context.Context, shortCode string) (shortenercontract.StatsResponse, error)
	ListURLs(ctx context.Context) ([]shortenercontract.URLEntry, error)
	DeleteURL(ctx context.Context, shortCode string) error
}

// AnalyticsPort is the driven port for the analytics module.
type AnalyticsPort interface {
	GetSummary(ctx context.Context) (analyticscontract.Summary, error)
	GetRecentLogs(ctx context.Context, limit int) ([]analyticscontract.AccessLog, error)
}
