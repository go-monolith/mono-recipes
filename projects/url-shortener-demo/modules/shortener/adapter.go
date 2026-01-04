package shortener

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-monolith/mono"
)

// ShortenerAdapterPort defines the interface for interacting with the shortener module.
// Consumers should use this interface instead of directly referencing the Module.
type ShortenerAdapterPort interface {
	ShortenURL(ctx context.Context, req ShortenRequest) (*ShortenResponse, error)
	ResolveURL(ctx context.Context, shortCode, userAgent, ipAddress string) (string, error)
	GetStats(ctx context.Context, shortCode string) (*StatsResponse, error)
	ListURLs(ctx context.Context) ([]URLEntry, error)
	DeleteURL(ctx context.Context, shortCode string) error
}

// shortenerAdapter implements ShortenerAdapterPort using the service container.
type shortenerAdapter struct {
	container mono.ServiceContainer
}

// NewShortenerAdapter creates a new adapter for the shortener service.
func NewShortenerAdapter(container mono.ServiceContainer) ShortenerAdapterPort {
	return &shortenerAdapter{
		container: container,
	}
}

// ShortenURL creates a new shortened URL.
func (a *shortenerAdapter) ShortenURL(ctx context.Context, req ShortenRequest) (*ShortenResponse, error) {
	client, err := a.container.GetRequestReplyService("shorten-url")
	if err != nil {
		return nil, fmt.Errorf("failed to get shorten-url service: %w", err)
	}

	reqData, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, reqData)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Check for error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &errResp); err == nil && errResp.Error != "" {
		return nil, mapServiceError(fmt.Errorf("%s", errResp.Error))
	}

	var response ShortenResponse
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// ResolveURL resolves a short code to its original URL and tracks the access.
func (a *shortenerAdapter) ResolveURL(ctx context.Context, shortCode, userAgent, ipAddress string) (string, error) {
	client, err := a.container.GetRequestReplyService("resolve-url")
	if err != nil {
		return "", fmt.Errorf("failed to get resolve-url service: %w", err)
	}

	req := struct {
		ShortCode string `json:"short_code"`
		UserAgent string `json:"user_agent,omitempty"`
		IPAddress string `json:"ip_address,omitempty"`
	}{
		ShortCode: shortCode,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	reqData, err := json.Marshal(&req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, reqData)
	if err != nil {
		return "", mapServiceError(err)
	}

	// Check for error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &errResp); err == nil && errResp.Error != "" {
		return "", mapServiceError(fmt.Errorf("%s", errResp.Error))
	}

	var response struct {
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.OriginalURL, nil
}

// GetStats retrieves statistics for a shortened URL.
func (a *shortenerAdapter) GetStats(ctx context.Context, shortCode string) (*StatsResponse, error) {
	client, err := a.container.GetRequestReplyService("get-stats")
	if err != nil {
		return nil, fmt.Errorf("failed to get get-stats service: %w", err)
	}

	req := struct {
		ShortCode string `json:"short_code"`
	}{
		ShortCode: shortCode,
	}

	reqData, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, reqData)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Check for error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &errResp); err == nil && errResp.Error != "" {
		return nil, mapServiceError(fmt.Errorf("%s", errResp.Error))
	}

	var response StatsResponse
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// ListURLs returns all active shortened URLs.
func (a *shortenerAdapter) ListURLs(ctx context.Context) ([]URLEntry, error) {
	client, err := a.container.GetRequestReplyService("list-urls")
	if err != nil {
		return nil, fmt.Errorf("failed to get list-urls service: %w", err)
	}

	resp, err := client.Call(ctx, []byte{})
	if err != nil {
		return nil, fmt.Errorf("list-urls service call failed: %w", err)
	}

	// Check for error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &errResp); err == nil && errResp.Error != "" {
		return nil, mapServiceError(fmt.Errorf("%s", errResp.Error))
	}

	var response []URLEntry
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

// DeleteURL removes a shortened URL.
func (a *shortenerAdapter) DeleteURL(ctx context.Context, shortCode string) error {
	client, err := a.container.GetRequestReplyService("delete-url")
	if err != nil {
		return fmt.Errorf("failed to get delete-url service: %w", err)
	}

	req := struct {
		ShortCode string `json:"short_code"`
	}{
		ShortCode: shortCode,
	}

	reqData, err := json.Marshal(&req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := client.Call(ctx, reqData)
	if err != nil {
		return mapServiceError(err)
	}

	// Check for error response
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(resp.Data, &errResp); err == nil && errResp.Error != "" {
		return mapServiceError(fmt.Errorf("%s", errResp.Error))
	}

	return nil
}

// mapServiceError converts service errors back to sentinel errors
// by checking the error message content. This is necessary because
// errors lose their type information when sent over NATS.
func mapServiceError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "url not found") || strings.Contains(errMsg, "not found") {
		return ErrURLNotFound
	}
	if strings.Contains(errMsg, "invalid short code") {
		return ErrInvalidShortCode
	}
	if strings.Contains(errMsg, "invalid url") {
		return ErrInvalidURL
	}
	if strings.Contains(errMsg, "expired") {
		return ErrURLExpired
	}

	return err
}
