package files

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ObjectStore defines the interface for file storage operations.
type ObjectStore interface {
	Put(ctx context.Context, name string, data []byte, contentType string) (*ObjectInfo, error)
	Get(ctx context.Context, name string) ([]byte, *ObjectInfo, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*ObjectInfo, error)
	GetInfo(ctx context.Context, name string) (*ObjectInfo, error)
}

// ObjectInfo represents metadata about a stored object.
type ObjectInfo struct {
	Name        string
	Size        uint64
	ContentType string
	ModTime     time.Time
}

// getContentType extracts Content-Type from headers with a default fallback.
func getContentType(headers nats.Header) string {
	if headers != nil {
		if ct := headers.Get("Content-Type"); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}

// JetStreamObjectStore implements ObjectStore using NATS JetStream Object Store.
type JetStreamObjectStore struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	store      jetstream.ObjectStore
	bucketName string
}

// NewJetStreamObjectStore creates a new JetStream Object Store client.
func NewJetStreamObjectStore(natsURL, bucketName string) (*JetStreamObjectStore, error) {
	// Connect to NATS
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &JetStreamObjectStore{
		conn:       conn,
		js:         js,
		bucketName: bucketName,
	}, nil
}

// Init initializes the object store bucket.
func (s *JetStreamObjectStore) Init(ctx context.Context) error {
	// Try to get existing bucket first
	store, err := s.js.ObjectStore(ctx, s.bucketName)
	if err == nil {
		s.store = store
		return nil
	}

	// Create bucket if it doesn't exist
	store, err = s.js.CreateObjectStore(ctx, jetstream.ObjectStoreConfig{
		Bucket:      s.bucketName,
		Description: "File upload storage bucket",
	})
	if err != nil {
		return fmt.Errorf("failed to create object store bucket: %w", err)
	}

	s.store = store
	return nil
}

// Put stores a file in the object store.
func (s *JetStreamObjectStore) Put(ctx context.Context, name string, data []byte, contentType string) (*ObjectInfo, error) {
	meta := jetstream.ObjectMeta{
		Name: name,
		Headers: nats.Header{
			"Content-Type": []string{contentType},
		},
	}

	info, err := s.store.Put(ctx, meta, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to store object: %w", err)
	}

	return &ObjectInfo{
		Name:        info.Name,
		Size:        info.Size,
		ContentType: contentType,
		ModTime:     info.ModTime,
	}, nil
}

// Get retrieves a file from the object store.
func (s *JetStreamObjectStore) Get(ctx context.Context, name string) ([]byte, *ObjectInfo, error) {
	result, err := s.store.Get(ctx, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer result.Close()

	data, err := io.ReadAll(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read object data: %w", err)
	}

	info, err := result.Info()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return data, &ObjectInfo{
		Name:        info.Name,
		Size:        info.Size,
		ContentType: getContentType(info.Headers),
		ModTime:     info.ModTime,
	}, nil
}

// Delete removes a file from the object store.
func (s *JetStreamObjectStore) Delete(ctx context.Context, name string) error {
	if err := s.store.Delete(ctx, name); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// List returns all files in the object store.
func (s *JetStreamObjectStore) List(ctx context.Context) ([]*ObjectInfo, error) {
	infos, err := s.store.List(ctx)
	if err != nil {
		// ErrNoObjectsFound means empty bucket, which is not an error for our use case
		if err.Error() == "nats: no objects found" {
			return []*ObjectInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	objects := make([]*ObjectInfo, 0, len(infos))
	for _, info := range infos {
		objects = append(objects, &ObjectInfo{
			Name:        info.Name,
			Size:        info.Size,
			ContentType: getContentType(info.Headers),
			ModTime:     info.ModTime,
		})
	}

	return objects, nil
}

// GetInfo retrieves metadata about a file without downloading its content.
func (s *JetStreamObjectStore) GetInfo(ctx context.Context, name string) (*ObjectInfo, error) {
	info, err := s.store.GetInfo(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &ObjectInfo{
		Name:        info.Name,
		Size:        info.Size,
		ContentType: getContentType(info.Headers),
		ModTime:     info.ModTime,
	}, nil
}

// IsConnected returns whether the NATS connection is active.
func (s *JetStreamObjectStore) IsConnected() bool {
	return s.conn != nil && s.conn.IsConnected()
}

// Close closes the NATS connection.
func (s *JetStreamObjectStore) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}
