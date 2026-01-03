package shortener

import (
	"context"
	"fmt"
	"net/url"
	"time"

	domain "github.com/example/url-shortener-demo/domain/url"
	"github.com/google/uuid"
)

// Service provides URL shortening operations.
type Service struct {
	kvStore    KVStore
	statsStore StatsStore
	baseURL    string
}

// NewService creates a new URL shortener service.
func NewService(kvStore KVStore, statsStore StatsStore, baseURL string) *Service {
	return &Service{
		kvStore:    kvStore,
		statsStore: statsStore,
		baseURL:    baseURL,
	}
}

// Shorten creates a new short URL.
func (s *Service) Shorten(ctx context.Context, originalURL string, customCode string, ttlSeconds int) (*domain.ShortURL, error) {
	// Validate URL
	if originalURL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	var shortCode string
	var err error

	if customCode != "" {
		// Validate custom code
		if !IsValidShortCode(customCode) {
			return nil, fmt.Errorf("invalid custom code: must be alphanumeric and max 20 characters")
		}

		// Check if custom code already exists
		exists, err := s.kvStore.Exists(ctx, customCode)
		if err != nil {
			return nil, fmt.Errorf("failed to check code availability: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("custom code already in use: %s", customCode)
		}
		shortCode = customCode
	} else {
		// Generate unique short code
		for i := 0; i < 10; i++ { // Max 10 attempts
			shortCode, err = GenerateShortCode(DefaultCodeLength)
			if err != nil {
				return nil, fmt.Errorf("failed to generate short code: %w", err)
			}

			exists, err := s.kvStore.Exists(ctx, shortCode)
			if err != nil {
				return nil, fmt.Errorf("failed to check code availability: %w", err)
			}
			if !exists {
				break
			}
			if i == 9 {
				return nil, fmt.Errorf("failed to generate unique short code after 10 attempts")
			}
		}
	}

	now := time.Now()
	shortURL := &domain.ShortURL{
		ID:          uuid.New().String(),
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		CreatedAt:   now,
	}

	// Set expiration if TTL is provided
	if ttlSeconds > 0 {
		expiresAt := now.Add(time.Duration(ttlSeconds) * time.Second)
		shortURL.ExpiresAt = &expiresAt
	}

	// Store the URL mapping
	if err := s.kvStore.Put(ctx, shortCode, shortURL); err != nil {
		return nil, fmt.Errorf("failed to store URL: %w", err)
	}

	// Initialize stats
	stats := &domain.URLStats{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		AccessCount: 0,
		CreatedAt:   now,
	}
	if err := s.statsStore.SetStats(ctx, shortCode, stats); err != nil {
		// Non-fatal: log but continue
		fmt.Printf("Warning: failed to initialize stats for %s: %v\n", shortCode, err)
	}

	return shortURL, nil
}

// Resolve retrieves the original URL for a short code.
func (s *Service) Resolve(ctx context.Context, shortCode string) (string, error) {
	if !IsValidShortCode(shortCode) {
		return "", fmt.Errorf("invalid short code format")
	}

	shortURL, err := s.kvStore.Get(ctx, shortCode)
	if err != nil {
		return "", err
	}

	return shortURL.OriginalURL, nil
}

// RecordAccess records an access event for a short code.
func (s *Service) RecordAccess(ctx context.Context, shortCode string) error {
	return s.statsStore.IncrementAccess(ctx, shortCode)
}

// GetStats retrieves statistics for a short code.
func (s *Service) GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error) {
	if !IsValidShortCode(shortCode) {
		return nil, fmt.Errorf("invalid short code format")
	}

	// First get the URL to ensure it exists
	shortURL, err := s.kvStore.Get(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// Get stats (or return default if not found)
	stats, err := s.statsStore.GetStats(ctx, shortCode)
	if err != nil {
		// Return stats with zero access count if not found
		return &domain.URLStats{
			ShortCode:   shortCode,
			OriginalURL: shortURL.OriginalURL,
			AccessCount: 0,
			CreatedAt:   shortURL.CreatedAt,
		}, nil
	}

	// Update original URL in case it's missing
	stats.OriginalURL = shortURL.OriginalURL

	return stats, nil
}

// GetFullShortURL returns the complete short URL with base URL.
func (s *Service) GetFullShortURL(shortCode string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shortCode)
}
