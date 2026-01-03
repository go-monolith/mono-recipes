package events

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// URLCreatedEvent is emitted when a new short URL is created.
type URLCreatedEvent struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// URLCreatedV1 is the typed event definition for URL creation.
// Subject: events.url.v1.url-created
var URLCreatedV1 = helper.EventDefinition[URLCreatedEvent](
	"url", "URLCreated", "v1",
)

// URLAccessedEvent is emitted when a short URL is accessed (redirected).
type URLAccessedEvent struct {
	ShortCode  string    `json:"short_code"`
	AccessedAt time.Time `json:"accessed_at"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Referer    string    `json:"referer,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
}

// URLAccessedV1 is the typed event definition for URL access.
// Subject: events.url.v1.url-accessed
var URLAccessedV1 = helper.EventDefinition[URLAccessedEvent](
	"url", "URLAccessed", "v1",
)
