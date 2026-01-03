// Package cache provides a Redis-based caching layer with cache-aside pattern.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides caching operations using Redis.
type Cache struct {
	client  *redis.Client
	prefix  string
	ttl     time.Duration
	stats   *Stats
}

// Stats tracks cache statistics.
type Stats struct {
	Hits       uint64 `json:"hits"`
	Misses     uint64 `json:"misses"`
	Sets       uint64 `json:"sets"`
	Deletes    uint64 `json:"deletes"`
	Errors     uint64 `json:"errors"`
}

// StatsSnapshot returns a snapshot of the current statistics.
type StatsSnapshot struct {
	Hits       uint64  `json:"hits"`
	Misses     uint64  `json:"misses"`
	Sets       uint64  `json:"sets"`
	Deletes    uint64  `json:"deletes"`
	Errors     uint64  `json:"errors"`
	HitRate    float64 `json:"hit_rate"`
	TotalGets  uint64  `json:"total_gets"`
}

// Config holds cache configuration.
type Config struct {
	RedisAddr string
	Prefix    string
	TTL       time.Duration
}

// DefaultConfig returns the default cache configuration.
func DefaultConfig() Config {
	return Config{
		RedisAddr: "localhost:6379",
		Prefix:    "cache:",
		TTL:       5 * time.Minute,
	}
}

// New creates a new cache instance.
func New(client *redis.Client, prefix string, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		prefix: prefix,
		ttl:    ttl,
		stats:  &Stats{},
	}
}

// Get retrieves a value from the cache.
// Returns the value and a boolean indicating if it was found (cache hit).
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	fullKey := c.prefix + key

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddUint64(&c.stats.Misses, 1)
			return false, nil // Cache miss
		}
		atomic.AddUint64(&c.stats.Errors, 1)
		return false, fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		atomic.AddUint64(&c.stats.Errors, 1)
		return false, fmt.Errorf("cache unmarshal error: %w", err)
	}

	atomic.AddUint64(&c.stats.Hits, 1)
	return true, nil
}

// Set stores a value in the cache with the default TTL.
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	return c.SetWithTTL(ctx, key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL.
func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.prefix + key

	data, err := json.Marshal(value)
	if err != nil {
		atomic.AddUint64(&c.stats.Errors, 1)
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		atomic.AddUint64(&c.stats.Errors, 1)
		return fmt.Errorf("cache set error: %w", err)
	}

	atomic.AddUint64(&c.stats.Sets, 1)
	return nil
}

// Delete removes a value from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		atomic.AddUint64(&c.stats.Errors, 1)
		return fmt.Errorf("cache delete error: %w", err)
	}

	atomic.AddUint64(&c.stats.Deletes, 1)
	return nil
}

// DeletePattern removes all keys matching a pattern.
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := c.prefix + pattern

	var cursor uint64
	var deletedCount int

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, fullPattern, 100).Result()
		if err != nil {
			atomic.AddUint64(&c.stats.Errors, 1)
			return fmt.Errorf("cache scan error: %w", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				atomic.AddUint64(&c.stats.Errors, 1)
				return fmt.Errorf("cache delete error: %w", err)
			}
			deletedCount += len(keys)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	atomic.AddUint64(&c.stats.Deletes, uint64(deletedCount))
	return nil
}

// GetStats returns the current cache statistics.
func (c *Cache) GetStats() StatsSnapshot {
	hits := atomic.LoadUint64(&c.stats.Hits)
	misses := atomic.LoadUint64(&c.stats.Misses)
	totalGets := hits + misses

	var hitRate float64
	if totalGets > 0 {
		hitRate = float64(hits) / float64(totalGets) * 100
	}

	return StatsSnapshot{
		Hits:      hits,
		Misses:    misses,
		Sets:      atomic.LoadUint64(&c.stats.Sets),
		Deletes:   atomic.LoadUint64(&c.stats.Deletes),
		Errors:    atomic.LoadUint64(&c.stats.Errors),
		HitRate:   hitRate,
		TotalGets: totalGets,
	}
}

// ResetStats resets all statistics counters.
func (c *Cache) ResetStats() {
	atomic.StoreUint64(&c.stats.Hits, 0)
	atomic.StoreUint64(&c.stats.Misses, 0)
	atomic.StoreUint64(&c.stats.Sets, 0)
	atomic.StoreUint64(&c.stats.Deletes, 0)
	atomic.StoreUint64(&c.stats.Errors, 0)
}

// Ping checks if the Redis connection is healthy.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis client connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

// GetClient returns the underlying Redis client.
func (c *Cache) GetClient() *redis.Client {
	return c.client
}
