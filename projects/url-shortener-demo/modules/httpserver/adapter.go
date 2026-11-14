package httpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-monolith/mono"
	monoerrors "github.com/go-monolith/mono/pkg/errors"

	analyticscontract "github.com/example/url-shortener-demo/modules/analytics/contract"
	shortenercontract "github.com/example/url-shortener-demo/modules/shortener/contract"
)

// shortenerAdapter is the driven adapter behind ShortenerPort: it calls the
// shortener's contracts with the container received in
// SetDependencyServiceContainer. It is the only place in the module that knows
// the call crosses a module boundary.
type shortenerAdapter struct{ container mono.ServiceContainer }

var _ ShortenerPort = shortenerAdapter{}

func (a shortenerAdapter) ShortenURL(ctx context.Context, req shortenercontract.ShortenRequest) (shortenercontract.ShortenResponse, error) {
	resp, err := shortenercontract.ShortenURL.Call(ctx, a.container, req)
	return resp, mapShortenerError(err)
}

func (a shortenerAdapter) ResolveURL(ctx context.Context, req shortenercontract.ResolveRequest) (string, error) {
	resp, err := shortenercontract.ResolveURL.Call(ctx, a.container, req)
	if err != nil {
		return "", mapShortenerError(err)
	}
	return resp.OriginalURL, nil
}

func (a shortenerAdapter) GetStats(ctx context.Context, shortCode string) (shortenercontract.StatsResponse, error) {
	resp, err := shortenercontract.GetStats.Call(ctx, a.container, shortenercontract.ShortCodeRequest{ShortCode: shortCode})
	return resp, mapShortenerError(err)
}

func (a shortenerAdapter) ListURLs(ctx context.Context) ([]shortenercontract.URLEntry, error) {
	resp, err := shortenercontract.ListURLs.Call(ctx, a.container, shortenercontract.ListURLsRequest{})
	return resp, mapShortenerError(err)
}

func (a shortenerAdapter) DeleteURL(ctx context.Context, shortCode string) error {
	_, err := shortenercontract.DeleteURL.Call(ctx, a.container, shortenercontract.ShortCodeRequest{ShortCode: shortCode})
	return mapShortenerError(err)
}

// mapShortenerError turns the RemoteError of a shortener handler back into
// the contract's sentinel error, keeping the handler's detail message, so the
// HTTP handlers can use errors.Is. Errors lose their Go type on the wire; only
// the message travels.
func mapShortenerError(err error) error {
	if err == nil {
		return nil
	}
	remote, ok := monoerrors.GetRemoteError(err)
	if !ok {
		return err
	}
	for _, sentinel := range shortenercontract.Errors {
		if detail, found := strings.CutPrefix(remote.Message, sentinel.Error()); found {
			return fmt.Errorf("%w%s", sentinel, detail)
		}
	}
	return errors.New(remote.Message)
}

// analyticsAdapter is the driven adapter behind AnalyticsPort.
type analyticsAdapter struct{ container mono.ServiceContainer }

var _ AnalyticsPort = analyticsAdapter{}

func (a analyticsAdapter) GetSummary(ctx context.Context) (analyticscontract.Summary, error) {
	return analyticscontract.GetSummary.Call(ctx, a.container, analyticscontract.SummaryRequest{})
}

func (a analyticsAdapter) GetRecentLogs(ctx context.Context, limit int) ([]analyticscontract.AccessLog, error) {
	return analyticscontract.GetLogs.Call(ctx, a.container, analyticscontract.LogsRequest{Limit: limit})
}
