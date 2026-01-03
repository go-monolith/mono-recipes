package api

import "time"

// ShortenRequest represents a URL shortening request.
type ShortenRequest struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
}

// ShortenResponse represents a URL shortening response.
type ShortenResponse struct {
	ID          string     `json:"id"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// StatsResponse represents a URL statistics response.
type StatsResponse struct {
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	AccessCount int64      `json:"access_count"`
	CreatedAt   time.Time  `json:"created_at"`
	LastAccess  *time.Time `json:"last_access,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}
