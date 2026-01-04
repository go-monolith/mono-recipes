package shortener

import "errors"

// Sentinel errors for shortener operations.
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
