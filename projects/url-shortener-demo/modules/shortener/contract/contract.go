// Package contract is the published port of the shortener module: the DTOs of
// its services, the typed service contracts other modules call them through,
// the events the module emits, and the errors its services report. It is the
// ONLY package of the shortener module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import (
	"errors"
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// ModuleName is the name of the module that provides these contracts.
const ModuleName = "shortener"

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// Errors reported by the shortener services. A service handler returns one of
// them (possibly wrapped with details); the framework sends the message back
// to the caller as a RemoteError, and consumers map it back to these values
// (see the http-server adapter) so callers can use errors.Is.
var (
	// ErrURLNotFound is returned when the requested short URL does not exist.
	ErrURLNotFound = errors.New("url not found")

	// ErrInvalidURL is returned when the provided URL is invalid.
	ErrInvalidURL = errors.New("invalid url")

	// ErrInvalidShortCode is returned when the short code format is invalid.
	ErrInvalidShortCode = errors.New("invalid short code")

	// ErrURLExpired is returned when the short URL has expired.
	ErrURLExpired = errors.New("url expired")
)

// Errors lists every error the shortener services report, for consumers that
// map remote error messages back to sentinel values.
var Errors = []error{ErrURLNotFound, ErrInvalidURL, ErrInvalidShortCode, ErrURLExpired}

// ---------------------------------------------------------------------------
// DTOs
// ---------------------------------------------------------------------------

// URLEntry represents a shortened URL.
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

// ResolveRequest is the request payload for resolving a short URL. UserAgent
// and IPAddress are copied into the URLAccessed event.
type ResolveRequest struct {
	ShortCode string `json:"short_code"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// ResolveResponse is the response after resolving a short URL.
type ResolveResponse struct {
	OriginalURL string `json:"original_url"`
}

// ShortCodeRequest is a request containing only a short code.
type ShortCodeRequest struct {
	ShortCode string `json:"short_code"`
}

// StatsResponse contains statistics for a shortened URL.
type StatsResponse struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessCount int64     `json:"access_count"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// ListURLsRequest is the (empty) request for listing URLs.
type ListURLsRequest struct{}

// DeleteResponse is the response after deleting a short URL.
type DeleteResponse struct {
	Message   string `json:"message"`
	ShortCode string `json:"short_code"`
}

// ---------------------------------------------------------------------------
// Service contracts (subjects services.shortener.<service>)
// ---------------------------------------------------------------------------

var (
	// ShortenURL creates a short URL and emits URLCreatedV1.
	ShortenURL = helper.RequestReplyService[ShortenRequest, ShortenResponse](ModuleName, "shorten-url")

	// ResolveURL resolves a short code, increments its access count and emits
	// URLAccessedV1.
	ResolveURL = helper.RequestReplyService[ResolveRequest, ResolveResponse](ModuleName, "resolve-url")

	// GetStats returns the statistics of a short URL.
	GetStats = helper.RequestReplyService[ShortCodeRequest, StatsResponse](ModuleName, "get-stats")

	// ListURLs returns every active short URL.
	ListURLs = helper.RequestReplyService[ListURLsRequest, []URLEntry](ModuleName, "list-urls")

	// DeleteURL removes a short URL.
	DeleteURL = helper.RequestReplyService[ShortCodeRequest, DeleteResponse](ModuleName, "delete-url")
)

// ---------------------------------------------------------------------------
// Events emitted by the shortener module
// ---------------------------------------------------------------------------

// URLCreatedEvent is published when a new URL is shortened.
type URLCreatedEvent struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	TTLSeconds  int64     `json:"ttl_seconds,omitempty"`
}

// URLAccessedEvent is published when a shortened URL is accessed.
type URLAccessedEvent struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	AccessedAt  time.Time `json:"accessed_at"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

var (
	// URLCreatedV1 is published when a new short URL is created
	// (subject events.shortener.v1.url-created).
	URLCreatedV1 = helper.EventDefinition[URLCreatedEvent](ModuleName, "URLCreated", "v1")

	// URLAccessedV1 is published when a short URL is accessed for redirect
	// (subject events.shortener.v1.url-accessed).
	URLAccessedV1 = helper.EventDefinition[URLAccessedEvent](ModuleName, "URLAccessed", "v1")
)
