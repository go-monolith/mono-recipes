package shortener

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/example/url-shortener-demo/domain/url"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// KVStore defines the interface for URL storage operations.
type KVStore interface {
	Put(ctx context.Context, shortCode string, url *domain.ShortURL) error
	Get(ctx context.Context, shortCode string) (*domain.ShortURL, error)
	Delete(ctx context.Context, shortCode string) error
	Exists(ctx context.Context, shortCode string) (bool, error)
}

// StatsStore defines the interface for URL statistics storage.
type StatsStore interface {
	IncrementAccess(ctx context.Context, shortCode string) error
	GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error)
	SetStats(ctx context.Context, shortCode string, stats *domain.URLStats) error
}

// JetStreamKVStore implements KVStore and StatsStore using NATS JetStream KV.
type JetStreamKVStore struct {
	conn        *nats.Conn
	js          jetstream.JetStream
	urlBucket   jetstream.KeyValue
	statsBucket jetstream.KeyValue
}

// NewJetStreamKVStore creates a new JetStream KV store client.
func NewJetStreamKVStore(natsURL string) (*JetStreamKVStore, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &JetStreamKVStore{
		conn: conn,
		js:   js,
	}, nil
}

// Init initializes the KV buckets.
func (s *JetStreamKVStore) Init(ctx context.Context) error {
	// Create or get URL bucket
	urlBucket, err := s.getOrCreateBucket(ctx, "urls", "URL short code to original URL mappings")
	if err != nil {
		return fmt.Errorf("failed to create urls bucket: %w", err)
	}
	s.urlBucket = urlBucket

	// Create or get stats bucket
	statsBucket, err := s.getOrCreateBucket(ctx, "url-stats", "URL access statistics")
	if err != nil {
		return fmt.Errorf("failed to create stats bucket: %w", err)
	}
	s.statsBucket = statsBucket

	return nil
}

func (s *JetStreamKVStore) getOrCreateBucket(ctx context.Context, name, description string) (jetstream.KeyValue, error) {
	bucket, err := s.js.KeyValue(ctx, name)
	if err == nil {
		return bucket, nil
	}

	bucket, err = s.js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      name,
		Description: description,
	})
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// Put stores a short URL mapping.
func (s *JetStreamKVStore) Put(ctx context.Context, shortCode string, url *domain.ShortURL) error {
	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("failed to marshal URL: %w", err)
	}

	if _, err := s.urlBucket.Put(ctx, shortCode, data); err != nil {
		return fmt.Errorf("failed to store URL: %w", err)
	}

	return nil
}

// Get retrieves a short URL mapping.
func (s *JetStreamKVStore) Get(ctx context.Context, shortCode string) (*domain.ShortURL, error) {
	entry, err := s.urlBucket.Get(ctx, shortCode)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, fmt.Errorf("short code not found: %s", shortCode)
		}
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	var url domain.ShortURL
	if err := json.Unmarshal(entry.Value(), &url); err != nil {
		return nil, fmt.Errorf("failed to unmarshal URL: %w", err)
	}

	// Check if URL has expired
	if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("short code has expired: %s", shortCode)
	}

	return &url, nil
}

// Delete removes a short URL mapping.
func (s *JetStreamKVStore) Delete(ctx context.Context, shortCode string) error {
	if err := s.urlBucket.Delete(ctx, shortCode); err != nil {
		return fmt.Errorf("failed to delete URL: %w", err)
	}
	return nil
}

// Exists checks if a short code exists.
func (s *JetStreamKVStore) Exists(ctx context.Context, shortCode string) (bool, error) {
	_, err := s.urlBucket.Get(ctx, shortCode)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check URL existence: %w", err)
	}
	return true, nil
}

// IncrementAccess increments the access count for a short code.
func (s *JetStreamKVStore) IncrementAccess(ctx context.Context, shortCode string) error {
	stats, err := s.GetStats(ctx, shortCode)
	if err != nil {
		// Initialize stats if not found
		stats = &domain.URLStats{
			ShortCode:   shortCode,
			AccessCount: 0,
			CreatedAt:   time.Now(),
		}
	}

	stats.AccessCount++
	now := time.Now()
	stats.LastAccess = &now

	return s.SetStats(ctx, shortCode, stats)
}

// GetStats retrieves statistics for a short code.
func (s *JetStreamKVStore) GetStats(ctx context.Context, shortCode string) (*domain.URLStats, error) {
	entry, err := s.statsBucket.Get(ctx, shortCode)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, fmt.Errorf("stats not found for: %s", shortCode)
		}
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	var stats domain.URLStats
	if err := json.Unmarshal(entry.Value(), &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}

// SetStats stores statistics for a short code.
func (s *JetStreamKVStore) SetStats(ctx context.Context, shortCode string, stats *domain.URLStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if _, err := s.statsBucket.Put(ctx, shortCode, data); err != nil {
		return fmt.Errorf("failed to store stats: %w", err)
	}

	return nil
}

// IsConnected returns whether the NATS connection is active.
func (s *JetStreamKVStore) IsConnected() bool {
	return s.conn != nil && s.conn.IsConnected()
}

// Close closes the NATS connection.
func (s *JetStreamKVStore) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}
