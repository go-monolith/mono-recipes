package shortener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"
	"github.com/jaevor/go-nanoid"
)

// shortCodeAlphabet is the alphabet used for generating short codes.
// Uses base62 characters (a-z, A-Z, 0-9) for URL-safe codes.
const shortCodeAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// shortCodeLength is the length of generated short codes.
const shortCodeLength = 8

// shortCodePattern validates the short code format.
var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9]{1,16}$`)

// Service provides URL shortening operations using the kv-jetstream plugin.
type Service struct {
	bucket    kvjetstream.KVStoragePort
	baseURL   string
	generateID func() string
}

// NewService creates a new shortener service with the given KV bucket.
func NewService(bucket kvjetstream.KVStoragePort, baseURL string) (*Service, error) {
	// Create nanoid generator with custom alphabet
	generator, err := nanoid.CustomASCII(shortCodeAlphabet, shortCodeLength)
	if err != nil {
		return nil, fmt.Errorf("failed to create ID generator: %w", err)
	}

	return &Service{
		bucket:    bucket,
		baseURL:   baseURL,
		generateID: generator,
	}, nil
}

// validateURL checks if the provided URL is valid and uses http/https scheme.
func validateURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidURL, err.Error())
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https", ErrInvalidURL)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	return nil
}

// validateShortCode checks if the short code format is valid.
func validateShortCode(code string) error {
	if code == "" {
		return ErrInvalidShortCode
	}
	if !shortCodePattern.MatchString(code) {
		return fmt.Errorf("%w: %s", ErrInvalidShortCode, code)
	}
	return nil
}

// ShortenURL creates a new shortened URL.
func (s *Service) ShortenURL(ctx context.Context, req ShortenRequest) (*ShortenResponse, error) {
	// Validate the URL
	if err := validateURL(req.URL); err != nil {
		return nil, err
	}

	// Generate unique short code
	shortCode := s.generateID()

	now := time.Now()
	entry := URLEntry{
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		CreatedAt:   now,
		AccessCount: 0,
	}

	// Calculate TTL if provided
	var ttl time.Duration
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
		entry.ExpiresAt = now.Add(ttl)
	}

	// Serialize and store
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry: %w", err)
	}

	err = s.bucket.Set(shortCode, data, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to store URL: %w", err)
	}

	return &ShortenResponse{
		ShortCode:   shortCode,
		ShortURL:    s.baseURL + "/" + shortCode,
		OriginalURL: req.URL,
		CreatedAt:   entry.CreatedAt,
		ExpiresAt:   entry.ExpiresAt,
	}, nil
}

// GetOriginalURL retrieves the original URL for a short code without incrementing access count.
func (s *Service) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	// Validate short code format
	if err := validateShortCode(shortCode); err != nil {
		return "", err
	}

	data, err := s.bucket.Get(shortCode)
	if err != nil {
		if errors.Is(err, kvjetstream.ErrKeyNotFound) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	var entry URLEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	return entry.OriginalURL, nil
}

// ResolveAndTrack retrieves the original URL and increments the access count.
func (s *Service) ResolveAndTrack(ctx context.Context, shortCode string) (string, error) {
	// Validate short code format
	if err := validateShortCode(shortCode); err != nil {
		return "", err
	}

	// Get current entry with revision for optimistic locking
	kvEntry, err := s.bucket.GetEntry(shortCode)
	if err != nil {
		if errors.Is(err, kvjetstream.ErrKeyNotFound) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("failed to get URL: %w", err)
	}

	var entry URLEntry
	if err := json.Unmarshal(kvEntry.Value, &entry); err != nil {
		return "", fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	// Increment access count
	entry.AccessCount++

	// Update with revision check (optimistic locking)
	newData, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Calculate remaining TTL if entry has expiration
	var ttl time.Duration
	if !entry.ExpiresAt.IsZero() {
		ttl = time.Until(entry.ExpiresAt)
		if ttl < 0 {
			return "", ErrURLExpired
		}
	}

	// Try to update, ignore revision mismatch (access count is best-effort)
	_, _ = s.bucket.Update(shortCode, newData, ttl, kvEntry.Revision)

	return entry.OriginalURL, nil
}

// GetStats retrieves statistics for a shortened URL.
func (s *Service) GetStats(ctx context.Context, shortCode string) (*StatsResponse, error) {
	// Validate short code format
	if err := validateShortCode(shortCode); err != nil {
		return nil, err
	}

	data, err := s.bucket.Get(shortCode)
	if err != nil {
		if errors.Is(err, kvjetstream.ErrKeyNotFound) {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	// Check for empty data (can happen with deleted keys in some KV implementations)
	if len(data) == 0 {
		return nil, ErrURLNotFound
	}

	var entry URLEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	return &StatsResponse{
		ShortCode:   entry.ShortCode,
		OriginalURL: entry.OriginalURL,
		AccessCount: entry.AccessCount,
		CreatedAt:   entry.CreatedAt,
		ExpiresAt:   entry.ExpiresAt,
	}, nil
}

// DeleteURL removes a shortened URL.
func (s *Service) DeleteURL(ctx context.Context, shortCode string) error {
	// Validate short code format
	if err := validateShortCode(shortCode); err != nil {
		return err
	}

	err := s.bucket.Delete(shortCode)
	if err != nil {
		if errors.Is(err, kvjetstream.ErrKeyNotFound) {
			return ErrURLNotFound
		}
		return fmt.Errorf("failed to delete URL: %w", err)
	}

	return nil
}

// ListURLs returns all active shortened URLs.
func (s *Service) ListURLs(ctx context.Context) ([]URLEntry, error) {
	keys, err := s.bucket.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var entries []URLEntry
	for _, key := range keys {
		data, err := s.bucket.Get(key)
		if err != nil {
			// Skip keys that may have expired
			continue
		}

		var entry URLEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
