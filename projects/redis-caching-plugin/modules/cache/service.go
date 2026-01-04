// Package cache provides a caching layer using the mono.Storage interface.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono/pkg/storage"
)

// CacheService defines the high-level caching operations used by consumers.
// This interface abstracts away the underlying storage implementation.
type CacheService interface {
	// Get retrieves a value from the cache and unmarshals it into dest.
	// Returns true if the key was found (cache hit), false otherwise.
	Get(ctx context.Context, key string, dest any) (bool, error)

	// Set stores a value in the cache with the default TTL.
	// The value is JSON-marshaled before storage.
	Set(ctx context.Context, key string, value any) error

	// SetWithTTL stores a value in the cache with a custom TTL.
	SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error

	// Delete removes a single key from the cache.
	Delete(ctx context.Context, key string) error

	// InvalidateAll clears ALL keys from the entire Redis database.
	// WARNING: This affects all keys in Redis, not just keys with this service's prefix.
	// Use with caution in shared Redis environments.
	InvalidateAll(ctx context.Context) error

	// Close closes the underlying storage connection.
	Close() error
}

// cacheService implements CacheService using the Storage interface.
type cacheService struct {
	storage storage.Storage
	prefix  string
	ttl     time.Duration
}

// NewCacheService creates a new CacheService wrapping the provided storage.
func NewCacheService(s storage.Storage, prefix string, ttl time.Duration) CacheService {
	return &cacheService{
		storage: s,
		prefix:  prefix,
		ttl:     ttl,
	}
}

// Get retrieves a value from the cache.
func (c *cacheService) Get(ctx context.Context, key string, dest any) (bool, error) {
	fullKey := c.prefix + key

	data, err := c.storage.GetWithContext(ctx, fullKey)
	if err != nil {
		return false, fmt.Errorf("cache get error: %w", err)
	}

	// nil or empty means key not found (cache miss)
	if len(data) == 0 {
		log.Printf("[cache] Cache Miss! key=%s", fullKey)
		return false, nil
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("cache unmarshal error: %w", err)
	}

	log.Printf("[cache] Cache Hit! key=%s", fullKey)
	return true, nil
}

// Set stores a value with the default TTL.
func (c *cacheService) Set(ctx context.Context, key string, value any) error {
	return c.SetWithTTL(ctx, key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL.
func (c *cacheService) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	fullKey := c.prefix + key

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := c.storage.SetWithContext(ctx, fullKey, data, ttl); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	return nil
}

// Delete removes a single key.
func (c *cacheService) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	if err := c.storage.DeleteWithContext(ctx, fullKey); err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}

	return nil
}

// InvalidateAll clears all keys from the entire Redis database.
// WARNING: This affects all keys in Redis, not just keys with this service's prefix.
func (c *cacheService) InvalidateAll(ctx context.Context) error {
	if err := c.storage.ResetWithContext(ctx); err != nil {
		return fmt.Errorf("cache reset error: %w", err)
	}
	return nil
}

// Close closes the underlying storage.
func (c *cacheService) Close() error {
	return c.storage.Close()
}
