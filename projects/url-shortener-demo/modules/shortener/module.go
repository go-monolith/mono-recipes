package shortener

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/example/url-shortener-demo/events"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ShortenerModule provides URL shortening services using NATS JetStream KV.
type ShortenerModule struct {
	store    *JetStreamKVStore
	service  *Service
	eventBus mono.EventBus
	natsURL  string
	baseURL  string
}

// Compile-time interface checks.
var _ mono.Module = (*ShortenerModule)(nil)
var _ mono.ServiceProviderModule = (*ShortenerModule)(nil)
var _ mono.EventEmitterModule = (*ShortenerModule)(nil)
var _ mono.HealthCheckableModule = (*ShortenerModule)(nil)

// NewModule creates a new ShortenerModule.
func NewModule() *ShortenerModule {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return &ShortenerModule{
		natsURL: natsURL,
		baseURL: baseURL,
	}
}

// Name returns the module name.
func (m *ShortenerModule) Name() string {
	return "shortener"
}

// SetEventBus is called by the framework to inject the event bus.
func (m *ShortenerModule) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents returns all event definitions this module can emit.
func (m *ShortenerModule) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		events.URLCreatedV1.ToBase(),
		events.URLAccessedV1.ToBase(),
	}
}

// RegisterServices registers request-reply services in the service container.
func (m *ShortenerModule) RegisterServices(container mono.ServiceContainer) error {
	// Register shorten-url service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"shorten-url",
		json.Unmarshal,
		json.Marshal,
		m.shortenURL,
	); err != nil {
		return fmt.Errorf("failed to register shorten-url service: %w", err)
	}

	// Register resolve-url service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"resolve-url",
		json.Unmarshal,
		json.Marshal,
		m.resolveURL,
	); err != nil {
		return fmt.Errorf("failed to register resolve-url service: %w", err)
	}

	// Register get-stats service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"get-stats",
		json.Unmarshal,
		json.Marshal,
		m.getStats,
	); err != nil {
		return fmt.Errorf("failed to register get-stats service: %w", err)
	}

	// Register record-access service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"record-access",
		json.Unmarshal,
		json.Marshal,
		m.recordAccess,
	); err != nil {
		return fmt.Errorf("failed to register record-access service: %w", err)
	}

	log.Printf("[shortener] Registered services: shorten-url, resolve-url, get-stats, record-access")
	return nil
}

// Start initializes the module and connects to NATS JetStream.
func (m *ShortenerModule) Start(ctx context.Context) error {
	var err error
	m.store, err = NewJetStreamKVStore(m.natsURL)
	if err != nil {
		return fmt.Errorf("failed to create KV store: %w", err)
	}

	if err := m.store.Init(ctx); err != nil {
		m.store.Close()
		return fmt.Errorf("failed to initialize KV store: %w", err)
	}

	m.service = NewService(m.store, m.store, m.baseURL)

	log.Printf("[shortener] Module started (NATS: %s, base URL: %s)", m.natsURL, m.baseURL)
	return nil
}

// Stop shuts down the module.
func (m *ShortenerModule) Stop(_ context.Context) error {
	if m.store != nil {
		m.store.Close()
	}
	log.Println("[shortener] Module stopped")
	return nil
}

// Health returns the health status of the module.
func (m *ShortenerModule) Health(_ context.Context) mono.HealthStatus {
	healthy := m.store != nil && m.store.IsConnected()
	message := "connected"
	if !healthy {
		message = "disconnected"
	}
	return mono.HealthStatus{
		Healthy: healthy,
		Message: message,
		Details: map[string]any{
			"nats_url": m.natsURL,
			"base_url": m.baseURL,
		},
	}
}

// shortenURL handles the shorten-url service request.
func (m *ShortenerModule) shortenURL(ctx context.Context, req ShortenRequest, _ *mono.Msg) (ShortenResponse, error) {
	shortURL, err := m.service.Shorten(ctx, req.URL, req.CustomCode, req.TTLSeconds)
	if err != nil {
		return ShortenResponse{}, err
	}

	// Emit URLCreated event
	if m.eventBus != nil {
		event := events.URLCreatedEvent{
			ShortCode:   shortURL.ShortCode,
			OriginalURL: shortURL.OriginalURL,
			CreatedAt:   shortURL.CreatedAt,
		}
		if err := events.URLCreatedV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[shortener] Warning: failed to publish URLCreated event: %v", err)
		}
	}

	return ShortenResponse{
		ID:          shortURL.ID,
		ShortCode:   shortURL.ShortCode,
		ShortURL:    m.service.GetFullShortURL(shortURL.ShortCode),
		OriginalURL: shortURL.OriginalURL,
		CreatedAt:   shortURL.CreatedAt,
		ExpiresAt:   shortURL.ExpiresAt,
	}, nil
}

// resolveURL handles the resolve-url service request.
func (m *ShortenerModule) resolveURL(ctx context.Context, req ResolveRequest, _ *mono.Msg) (ResolveResponse, error) {
	originalURL, err := m.service.Resolve(ctx, req.ShortCode)
	if err != nil {
		return ResolveResponse{}, err
	}

	return ResolveResponse{
		OriginalURL: originalURL,
		ShortCode:   req.ShortCode,
	}, nil
}

// getStats handles the get-stats service request.
func (m *ShortenerModule) getStats(ctx context.Context, req GetStatsRequest, _ *mono.Msg) (GetStatsResponse, error) {
	stats, err := m.service.GetStats(ctx, req.ShortCode)
	if err != nil {
		return GetStatsResponse{}, err
	}

	return GetStatsResponse{
		ShortCode:   stats.ShortCode,
		OriginalURL: stats.OriginalURL,
		AccessCount: stats.AccessCount,
		CreatedAt:   stats.CreatedAt,
		LastAccess:  stats.LastAccess,
	}, nil
}

// recordAccess handles the record-access service request.
func (m *ShortenerModule) recordAccess(ctx context.Context, req RecordAccessRequest, _ *mono.Msg) (RecordAccessResponse, error) {
	if err := m.service.RecordAccess(ctx, req.ShortCode); err != nil {
		return RecordAccessResponse{Recorded: false}, err
	}

	// Emit URLAccessed event
	if m.eventBus != nil {
		event := events.URLAccessedEvent{
			ShortCode:  req.ShortCode,
			AccessedAt: time.Now(),
			UserAgent:  req.UserAgent,
			Referer:    req.Referer,
			IPAddress:  req.IPAddress,
		}
		if err := events.URLAccessedV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[shortener] Warning: failed to publish URLAccessed event: %v", err)
		}
	}

	return RecordAccessResponse{Recorded: true}, nil
}
