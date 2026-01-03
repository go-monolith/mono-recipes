package shortener

import "time"

// ShortenRequest represents a URL shortening request.
type ShortenRequest struct {
	URL       string `json:"url"`
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

// ResolveRequest represents a URL resolution request.
type ResolveRequest struct {
	ShortCode string `json:"short_code"`
}

// ResolveResponse represents a URL resolution response.
type ResolveResponse struct {
	OriginalURL string `json:"original_url"`
	ShortCode   string `json:"short_code"`
}

// GetStatsRequest represents a URL stats request.
type GetStatsRequest struct {
	ShortCode string `json:"short_code"`
}

// GetStatsResponse represents a URL stats response.
type GetStatsResponse struct {
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	AccessCount int64      `json:"access_count"`
	CreatedAt   time.Time  `json:"created_at"`
	LastAccess  *time.Time `json:"last_access,omitempty"`
}

// RecordAccessRequest represents a request to record an access event.
type RecordAccessRequest struct {
	ShortCode string `json:"short_code"`
	UserAgent string `json:"user_agent,omitempty"`
	Referer   string `json:"referer,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// RecordAccessResponse represents a response from recording an access event.
type RecordAccessResponse struct {
	Recorded bool `json:"recorded"`
}
