package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// AnalyticsAdapterPort defines the interface for interacting with the analytics module.
// Consumers should use this interface instead of directly referencing the Module.
type AnalyticsAdapterPort interface {
	GetSummary(ctx context.Context) (map[string]any, error)
	GetRecentLogs(ctx context.Context, limit int) ([]AccessLog, error)
}

// analyticsAdapter implements AnalyticsAdapterPort using the service container.
type analyticsAdapter struct {
	container mono.ServiceContainer
}

// NewAnalyticsAdapter creates a new adapter for the analytics service.
func NewAnalyticsAdapter(container mono.ServiceContainer) AnalyticsAdapterPort {
	return &analyticsAdapter{
		container: container,
	}
}

// GetSummary retrieves the analytics summary.
func (a *analyticsAdapter) GetSummary(ctx context.Context) (map[string]any, error) {
	client, err := a.container.GetRequestReplyService("get-analytics-summary")
	if err != nil {
		return nil, fmt.Errorf("failed to get get-analytics-summary service: %w", err)
	}

	resp, err := client.Call(ctx, []byte{})
	if err != nil {
		return nil, fmt.Errorf("get-analytics-summary service call failed: %w", err)
	}

	var response map[string]any
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

// GetRecentLogs retrieves recent access logs.
func (a *analyticsAdapter) GetRecentLogs(ctx context.Context, limit int) ([]AccessLog, error) {
	req := struct {
		Limit int `json:"limit"`
	}{
		Limit: limit,
	}

	var response []AccessLog
	err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"get-analytics-logs",
		json.Marshal,
		json.Unmarshal,
		&req,
		&response,
	)
	if err != nil {
		return nil, fmt.Errorf("get-analytics-logs service call failed: %w", err)
	}
	return response, nil
}
