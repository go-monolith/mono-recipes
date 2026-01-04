package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/redis/go-redis/v9"
)

// Middleware implements rate limiting as a mono.MiddlewareModule.
// It intercepts request-reply service registrations and wraps handlers
// to enforce per-client, per-service rate limits using Redis.
type Middleware struct {
	name    string
	config  Config
	client  *redis.Client
	limiter *Limiter
	logger  *slog.Logger
}

// Compile-time interface checks
var _ mono.Module = (*Middleware)(nil)
var _ mono.MiddlewareModule = (*Middleware)(nil)

// RateLimitError is returned when rate limit is exceeded.
type RateLimitError struct {
	Message   string    `json:"error"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
	Limit     int       `json:"limit"`
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// New creates a new rate limiting middleware.
func New(opts ...Option) (*Middleware, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	return &Middleware{
		name:   "rate-limit",
		config: config,
		logger: slog.Default(),
	}, nil
}

// Name returns the middleware name.
func (m *Middleware) Name() string {
	return m.name
}

// Start initializes the Redis connection.
func (m *Middleware) Start(ctx context.Context) error {
	m.client = redis.NewClient(&redis.Options{
		Addr:         m.config.RedisAddr,
		Password:     m.config.RedisPassword,
		DB:           m.config.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	// Test Redis connection
	if err := m.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis at %s: %w", m.config.RedisAddr, err)
	}

	m.limiter = NewLimiter(m.client, m.config.KeyPrefix)
	m.logger.Info("Rate limiting middleware started",
		"redis", m.config.RedisAddr,
		"default_limit", m.config.DefaultLimit,
		"default_window", m.config.DefaultWindow)

	return nil
}

// Stop closes the Redis connection.
func (m *Middleware) Stop(ctx context.Context) error {
	if m.client != nil {
		if err := m.client.Close(); err != nil {
			m.logger.Error("Failed to close Redis connection", "error", err)
			return err
		}
	}
	m.logger.Info("Rate limiting middleware stopped")
	return nil
}

// OnModuleLifecycle passes through module lifecycle events unchanged.
func (m *Middleware) OnModuleLifecycle(
	_ context.Context,
	event types.ModuleLifecycleEvent,
) types.ModuleLifecycleEvent {
	return event
}

// OnServiceRegistration wraps request-reply handlers with rate limiting.
func (m *Middleware) OnServiceRegistration(
	_ context.Context,
	reg types.ServiceRegistration,
) types.ServiceRegistration {
	// Only wrap request-reply services
	if reg.Type != types.ServiceTypeRequestReply || reg.RequestHandler == nil {
		return reg
	}

	serviceName := reg.Name
	original := reg.RequestHandler

	// Get limit configuration for this service
	limit, window := m.getLimitForService(serviceName)

	m.logger.Debug("Wrapping service with rate limiting",
		"service", serviceName,
		"limit", limit,
		"window", window)

	// Wrap the handler with rate limiting
	reg.RequestHandler = func(ctx context.Context, req *types.Msg) ([]byte, error) {
		// Extract client ID from request headers
		clientID := m.extractClientID(req)

		// Create rate limit key: service:clientID
		key := fmt.Sprintf("%s:%s", serviceName, clientID)

		// Check rate limit
		result, err := m.limiter.Allow(ctx, key, limit, window)
		if err != nil {
			m.logger.Error("Rate limit check failed",
				"service", serviceName,
				"client_id", clientID,
				"error", err)
			// On Redis error, allow the request (fail-open)
			return original(ctx, req)
		}

		if !result.Allowed {
			m.logger.Warn("Rate limit exceeded",
				"service", serviceName,
				"client_id", clientID,
				"limit", result.Limit,
				"reset_at", result.ResetAt)

			errResp := &RateLimitError{
				Message:   fmt.Sprintf("rate limit exceeded for service %s", serviceName),
				Remaining: result.Remaining,
				ResetAt:   result.ResetAt,
				Limit:     result.Limit,
			}

			respBytes, err := json.Marshal(errResp)
			if err != nil {
				m.logger.Error("Failed to marshal rate limit error", "error", err)
				return nil, errResp
			}
			return respBytes, errResp
		}

		m.logger.Debug("Request allowed",
			"service", serviceName,
			"client_id", clientID,
			"remaining", result.Remaining)

		return original(ctx, req)
	}

	return reg
}

// OnConfigurationChange passes through configuration changes unchanged.
func (m *Middleware) OnConfigurationChange(
	_ context.Context,
	event types.ConfigurationEvent,
) types.ConfigurationEvent {
	return event
}

// OnOutgoingMessage passes through outgoing messages unchanged.
func (m *Middleware) OnOutgoingMessage(
	octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
	return octx
}

// OnEventConsumerRegistration passes through event consumer registrations unchanged.
func (m *Middleware) OnEventConsumerRegistration(
	_ context.Context,
	entry types.EventConsumerEntry,
) types.EventConsumerEntry {
	return entry
}

// OnEventStreamConsumerRegistration passes through event stream consumer registrations unchanged.
func (m *Middleware) OnEventStreamConsumerRegistration(
	_ context.Context,
	entry types.EventStreamConsumerEntry,
) types.EventStreamConsumerEntry {
	return entry
}

// getLimitForService returns the rate limit configuration for a service.
func (m *Middleware) getLimitForService(serviceName string) (int, time.Duration) {
	if serviceLimit, ok := m.config.ServiceLimits[serviceName]; ok {
		return serviceLimit.Limit, serviceLimit.Window
	}
	return m.config.DefaultLimit, m.config.DefaultWindow
}

// maxClientIDLength limits client ID length to prevent abuse.
const maxClientIDLength = 128

// extractClientID extracts the client ID from request headers.
// It sanitizes and truncates the value to prevent abuse.
func (m *Middleware) extractClientID(req *types.Msg) string {
	if req.Header != nil {
		if values, ok := req.Header[m.config.ClientIDHeader]; ok && len(values) > 0 {
			clientID := values[0]
			// Truncate excessively long client IDs
			if len(clientID) > maxClientIDLength {
				clientID = clientID[:maxClientIDLength]
			}
			// Skip empty client IDs
			if clientID == "" {
				return m.config.FallbackClientID
			}
			return clientID
		}
	}
	return m.config.FallbackClientID
}
