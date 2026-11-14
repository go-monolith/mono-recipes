// Package contract is the published port of the analytics module: the DTOs of
// its services and the typed contracts other modules call them through. It is
// the ONLY package of the analytics module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// ModuleName is the name of the module that provides these contracts.
const ModuleName = "analytics"

// AccessLog represents a single access log entry.
type AccessLog struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessedAt  time.Time `json:"accessed_at"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

// SummaryRequest is the (empty) request for the analytics summary.
type SummaryRequest struct{}

// Summary is an overall analytics summary.
type Summary struct {
	URLsCreated   int64 `json:"urls_created"`
	URLsTracked   int   `json:"urls_tracked"`
	TotalAccesses int64 `json:"total_accesses"`
	AccessLogs    int   `json:"access_logs"`
}

// LogsRequest asks for the most recent access logs. Limit defaults to 100 and
// is capped at 1000 by the provider.
type LogsRequest struct {
	Limit int `json:"limit"`
}

var (
	// GetSummary returns the analytics summary
	// (subject services.analytics.get-analytics-summary).
	GetSummary = helper.RequestReplyService[SummaryRequest, Summary](ModuleName, "get-analytics-summary")

	// GetLogs returns the most recent access logs
	// (subject services.analytics.get-analytics-logs).
	GetLogs = helper.RequestReplyService[LogsRequest, []AccessLog](ModuleName, "get-analytics-logs")
)
