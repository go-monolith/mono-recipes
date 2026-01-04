// Example: Creating a Custom Middleware Module
//
// This example demonstrates:
// - Implementing the MiddlewareModule interface
// - Wrapping service handlers (decorator pattern)
// - Observing lifecycle events
// - Injecting headers into outgoing messages
// - Collecting metrics from handlers

package middlewaremodule

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
)

// ============================================================
// Custom Metrics Middleware Implementation
// ============================================================

// MetricsMiddleware collects request metrics from all service handlers.
type MetricsMiddleware struct {
	name         string
	requestCount atomic.Int64
	errorCount   atomic.Int64
	totalLatency atomic.Int64
	mu           sync.RWMutex
	serviceStats map[string]*ServiceMetrics
}

// ServiceMetrics holds per-service metrics.
type ServiceMetrics struct {
	RequestCount int64
	ErrorCount   int64
	TotalLatency int64
}

// Compile-time interface check
var _ types.MiddlewareModule = (*MetricsMiddleware)(nil)

// NewMetricsMiddleware creates a new metrics middleware.
func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		name:         "metrics",
		serviceStats: make(map[string]*ServiceMetrics),
	}
}

// ============================================================
// Module Interface Implementation
// ============================================================

func (m *MetricsMiddleware) Name() string {
	return m.name
}

func (m *MetricsMiddleware) Start(_ context.Context) error {
	slog.Info("Metrics middleware started")
	return nil
}

func (m *MetricsMiddleware) Stop(_ context.Context) error {
	slog.Info("Metrics middleware stopped",
		"total_requests", m.requestCount.Load(),
		"total_errors", m.errorCount.Load(),
		"avg_latency_ms", m.AverageLatencyMS())
	return nil
}

// ============================================================
// MiddlewareModule Hook Implementations
// ============================================================

// OnModuleLifecycle observes module lifecycle events.
func (m *MetricsMiddleware) OnModuleLifecycle(
	_ context.Context,
	event types.ModuleLifecycleEvent,
) types.ModuleLifecycleEvent {
	switch event.Type {
	case types.ModuleStartedEvent:
		slog.Info("Module started (observed by metrics)",
			"module", event.ModuleName,
			"startup_ms", event.Duration.Milliseconds())
	case types.ModuleStoppedEvent:
		if event.Error != nil {
			slog.Warn("Module stopped with error",
				"module", event.ModuleName,
				"error", event.Error)
		}
	}
	return event // Pass through unchanged (observer pattern)
}

// OnServiceRegistration wraps handlers to collect metrics.
func (m *MetricsMiddleware) OnServiceRegistration(
	_ context.Context,
	reg types.ServiceRegistration,
) types.ServiceRegistration {
	switch reg.Type {
	case types.ServiceTypeRequestReply:
		if reg.RequestHandler != nil {
			reg.RequestHandler = m.wrapRequestReplyHandler(
				reg.RequestHandler,
				reg.Name,
			)
		}

	case types.ServiceTypeQueueGroup:
		if len(reg.QueueHandlers) > 0 {
			wrappedPairs := make([]types.QGHP, len(reg.QueueHandlers))
			for i, pair := range reg.QueueHandlers {
				wrappedPairs[i] = types.QGHP{
					QueueGroup: pair.QueueGroup,
					Handler:    m.wrapQueueGroupHandler(pair.Handler, reg.Name),
				}
			}
			reg.QueueHandlers = wrappedPairs
		}

	case types.ServiceTypeStreamConsumer:
		if reg.StreamHandler != nil {
			reg.StreamHandler = m.wrapStreamConsumerHandler(
				reg.StreamHandler,
				reg.Name,
			)
		}
	}

	return reg
}

// OnConfigurationChange passes through configuration events.
func (m *MetricsMiddleware) OnConfigurationChange(
	_ context.Context,
	event types.ConfigurationEvent,
) types.ConfigurationEvent {
	return event // Pass through unchanged
}

// OnOutgoingMessage passes through outgoing messages.
func (m *MetricsMiddleware) OnOutgoingMessage(
	octx types.OutgoingMessageContext,
) types.OutgoingMessageContext {
	return octx // Pass through unchanged
}

// OnEventConsumerRegistration wraps event consumer handlers.
func (m *MetricsMiddleware) OnEventConsumerRegistration(
	_ context.Context,
	entry types.EventConsumerEntry,
) types.EventConsumerEntry {
	if entry.Handler != nil {
		entry.Handler = m.wrapEventConsumerHandler(
			entry.Handler,
			entry.EventDef.Name,
		)
	}
	return entry
}

// OnEventStreamConsumerRegistration wraps event stream consumer handlers.
func (m *MetricsMiddleware) OnEventStreamConsumerRegistration(
	_ context.Context,
	entry types.EventStreamConsumerEntry,
) types.EventStreamConsumerEntry {
	if entry.Handler != nil {
		entry.Handler = m.wrapEventStreamConsumerHandler(
			entry.Handler,
			entry.EventDef.Name,
		)
	}
	return entry
}

// ============================================================
// Handler Wrappers
// ============================================================

func (m *MetricsMiddleware) wrapRequestReplyHandler(
	original types.RequestReplyHandler,
	serviceName string,
) types.RequestReplyHandler {
	return func(ctx context.Context, req *types.Msg) ([]byte, error) {
		start := time.Now()
		resp, err := original(ctx, req)
		m.recordMetrics(serviceName, time.Since(start), err)
		return resp, err
	}
}

func (m *MetricsMiddleware) wrapQueueGroupHandler(
	original types.QueueGroupHandler,
	serviceName string,
) types.QueueGroupHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		start := time.Now()
		err := original(ctx, msg)
		m.recordMetrics(serviceName, time.Since(start), err)
		return err
	}
}

func (m *MetricsMiddleware) wrapStreamConsumerHandler(
	original types.StreamConsumerHandler,
	serviceName string,
) types.StreamConsumerHandler {
	return func(ctx context.Context, msgs []*types.Msg) error {
		start := time.Now()
		err := original(ctx, msgs)
		m.recordMetrics(serviceName, time.Since(start), err)
		return err
	}
}

func (m *MetricsMiddleware) wrapEventConsumerHandler(
	original types.EventConsumerHandler,
	eventName string,
) types.EventConsumerHandler {
	return func(ctx context.Context, msg *types.Msg) error {
		start := time.Now()
		err := original(ctx, msg)
		m.recordMetrics(eventName, time.Since(start), err)
		return err
	}
}

func (m *MetricsMiddleware) wrapEventStreamConsumerHandler(
	original types.EventStreamConsumerHandler,
	eventName string,
) types.EventStreamConsumerHandler {
	return func(ctx context.Context, msgs []*types.Msg) error {
		start := time.Now()
		err := original(ctx, msgs)
		m.recordMetrics(eventName, time.Since(start), err)
		return err
	}
}

// ============================================================
// Metrics Recording
// ============================================================

func (m *MetricsMiddleware) recordMetrics(serviceName string, latency time.Duration, err error) {
	m.requestCount.Add(1)
	m.totalLatency.Add(latency.Milliseconds())
	if err != nil {
		m.errorCount.Add(1)
	}

	// Record per-service metrics
	m.mu.Lock()
	stats, ok := m.serviceStats[serviceName]
	if !ok {
		stats = &ServiceMetrics{}
		m.serviceStats[serviceName] = stats
	}
	stats.RequestCount++
	stats.TotalLatency += latency.Milliseconds()
	if err != nil {
		stats.ErrorCount++
	}
	m.mu.Unlock()
}

// ============================================================
// Public Metrics API
// ============================================================

// RequestCount returns total request count.
func (m *MetricsMiddleware) RequestCount() int64 {
	return m.requestCount.Load()
}

// ErrorCount returns total error count.
func (m *MetricsMiddleware) ErrorCount() int64 {
	return m.errorCount.Load()
}

// AverageLatencyMS returns average latency in milliseconds.
func (m *MetricsMiddleware) AverageLatencyMS() float64 {
	count := m.requestCount.Load()
	if count == 0 {
		return 0
	}
	return float64(m.totalLatency.Load()) / float64(count)
}

// ServiceStats returns per-service metrics.
func (m *MetricsMiddleware) ServiceStats() map[string]ServiceMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]ServiceMetrics, len(m.serviceStats))
	for name, stats := range m.serviceStats {
		result[name] = ServiceMetrics{
			RequestCount: stats.RequestCount,
			ErrorCount:   stats.ErrorCount,
			TotalLatency: stats.TotalLatency,
		}
	}
	return result
}

// ============================================================
// Sample Module Using the Middleware
// ============================================================

type GreetingModule struct {
	container types.ServiceContainer
}

var _ mono.ServiceProviderModule = (*GreetingModule)(nil)

func NewGreetingModule() *GreetingModule {
	return &GreetingModule{}
}

func (m *GreetingModule) Name() string { return "greeting" }

func (m *GreetingModule) Start(_ context.Context) error {
	slog.Info("Greeting module started")
	return nil
}

func (m *GreetingModule) Stop(_ context.Context) error {
	slog.Info("Greeting module stopped")
	return nil
}

func (m *GreetingModule) RegisterServices(container types.ServiceContainer) error {
	m.container = container
	return container.RegisterRequestReplyService("greet", m.handleGreet)
}

func (m *GreetingModule) handleGreet(_ context.Context, req *types.Msg) ([]byte, error) {
	name := string(req.Data)
	if name == "" {
		name = "World"
	}
	return []byte(fmt.Sprintf("Hello, %s!", name)), nil
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Custom Middleware Example ===")
	fmt.Println("Demonstrates: Creating a metrics collection middleware")
	fmt.Println()

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create and register middleware FIRST
	metricsMiddleware := NewMetricsMiddleware()
	if err := app.Register(metricsMiddleware); err != nil {
		log.Fatalf("Failed to register middleware: %v", err)
	}
	fmt.Println("Metrics middleware registered")

	// Register regular modules AFTER middleware
	greetingModule := NewGreetingModule()
	if err := app.Register(greetingModule); err != nil {
		log.Fatalf("Failed to register module: %v", err)
	}
	fmt.Println("Greeting module registered")

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Println()

	// Simulate some requests to collect metrics
	fmt.Println("=== Simulating Requests ===")

	greetClient, err := greetingModule.container.GetRequestReplyService("greet")
	if err != nil {
		log.Fatalf("Failed to get greeting service: %v", err)
	}

	// Make some requests
	names := []string{"Alice", "Bob", "Charlie", "", "Diana"}
	for _, name := range names {
		resp, err := greetClient.Call(ctx, []byte(name))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n", string(resp))
		}
	}

	fmt.Println()
	fmt.Println("=== Collected Metrics ===")
	fmt.Printf("Total Requests: %d\n", metricsMiddleware.RequestCount())
	fmt.Printf("Total Errors: %d\n", metricsMiddleware.ErrorCount())
	fmt.Printf("Average Latency: %.2f ms\n", metricsMiddleware.AverageLatencyMS())

	fmt.Println("\nPer-Service Metrics:")
	for name, stats := range metricsMiddleware.ServiceStats() {
		avgLatency := float64(0)
		if stats.RequestCount > 0 {
			avgLatency = float64(stats.TotalLatency) / float64(stats.RequestCount)
		}
		fmt.Printf("  %s: %d requests, %d errors, %.2f ms avg\n",
			name, stats.RequestCount, stats.ErrorCount, avgLatency)
	}

	// Wait for shutdown signal
	fmt.Println("\nPress Ctrl+C to shutdown...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}

	fmt.Println("App stopped successfully")
}
