package url

import (
	"time"
)

// ShortURL represents a shortened URL mapping.
type ShortURL struct {
	ID          string    `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// URLStats represents statistics for a shortened URL.
type URLStats struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessCount int64     `json:"access_count"`
	CreatedAt   time.Time `json:"created_at"`
	LastAccess  *time.Time `json:"last_access,omitempty"`
}
