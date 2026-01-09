package shortener

import "time"

// URLEntry represents a shortened URL stored in the KV store.
type URLEntry struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	AccessCount int64     `json:"access_count"`
}

// ShortenRequest is the request payload for creating a short URL.
type ShortenRequest struct {
	URL string `json:"url"`
	// Optional TTL in seconds (0 = never expires)
	TTLSeconds int64 `json:"ttl_seconds,omitempty"`
}

// ShortenResponse is the response after creating a short URL.
type ShortenResponse struct {
	ShortCode   string    `json:"short_code"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// StatsResponse contains statistics for a shortened URL.
type StatsResponse struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessCount int64     `json:"access_count"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// ShortCodeRequest is a request containing only a short code.
type ShortCodeRequest struct {
	ShortCode string `json:"short_code"`
}

// ResolveRequest is the request payload for resolving a short URL.
type ResolveRequest struct {
	ShortCode string `json:"short_code"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// ResolveResponse is the response after resolving a short URL.
type ResolveResponse struct {
	OriginalURL string `json:"original_url"`
}

// DeleteResponse is the response after deleting a short URL.
type DeleteResponse struct {
	Message   string `json:"message"`
	ShortCode string `json:"short_code"`
}

// URLAccessedEvent is published when a shortened URL is accessed.
type URLAccessedEvent struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessedAt  time.Time `json:"accessed_at"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

// URLCreatedEvent is published when a new URL is shortened.
type URLCreatedEvent struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	TTLSeconds  int64     `json:"ttl_seconds,omitempty"`
}
