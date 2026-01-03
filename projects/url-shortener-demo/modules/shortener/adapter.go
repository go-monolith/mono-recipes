package shortener

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ShortenerPort defines the interface for URL shortening operations from other modules.
type ShortenerPort interface {
	ShortenURL(ctx context.Context, url string, customCode string, ttlSeconds int) (*ShortenResponse, error)
	ResolveURL(ctx context.Context, shortCode string) (*ResolveResponse, error)
	GetStats(ctx context.Context, shortCode string) (*GetStatsResponse, error)
	RecordAccess(ctx context.Context, shortCode, userAgent, referer, ipAddress string) error
}

// shortenerAdapter wraps ServiceContainer for type-safe cross-module communication.
type shortenerAdapter struct {
	container mono.ServiceContainer
}

// NewShortenerAdapter creates a new adapter for shortener services.
func NewShortenerAdapter(container mono.ServiceContainer) ShortenerPort {
	if container == nil {
		panic("shortener adapter requires non-nil ServiceContainer")
	}
	return &shortenerAdapter{container: container}
}

// ShortenURL shortens a URL via the shorten-url service.
func (a *shortenerAdapter) ShortenURL(ctx context.Context, url string, customCode string, ttlSeconds int) (*ShortenResponse, error) {
	req := ShortenRequest{
		URL:        url,
		CustomCode: customCode,
		TTLSeconds: ttlSeconds,
	}
	var resp ShortenResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"shorten-url",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("shorten-url service call failed: %w", err)
	}
	return &resp, nil
}

// ResolveURL resolves a short code via the resolve-url service.
func (a *shortenerAdapter) ResolveURL(ctx context.Context, shortCode string) (*ResolveResponse, error) {
	req := ResolveRequest{ShortCode: shortCode}
	var resp ResolveResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"resolve-url",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("resolve-url service call failed: %w", err)
	}
	return &resp, nil
}

// GetStats retrieves stats via the get-stats service.
func (a *shortenerAdapter) GetStats(ctx context.Context, shortCode string) (*GetStatsResponse, error) {
	req := GetStatsRequest{ShortCode: shortCode}
	var resp GetStatsResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"get-stats",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("get-stats service call failed: %w", err)
	}
	return &resp, nil
}

// RecordAccess records an access event via the record-access service.
func (a *shortenerAdapter) RecordAccess(ctx context.Context, shortCode, userAgent, referer, ipAddress string) error {
	req := RecordAccessRequest{
		ShortCode: shortCode,
		UserAgent: userAgent,
		Referer:   referer,
		IPAddress: ipAddress,
	}
	var resp RecordAccessResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"record-access",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("record-access service call failed: %w", err)
	}
	return nil
}
