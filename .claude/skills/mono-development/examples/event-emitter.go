// Example: Event Emitter and Consumer
//
// This example demonstrates:
// - Defining typed events with helper.EventDefinition
// - Implementing EventEmitterModule
// - Implementing EventConsumerModule
// - Event discovery via EventRegistry
// - Fire-and-forget event publishing

package eventemitter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ============================================================
// Event Definitions
// ============================================================

// OrderCreatedEvent is the event payload
type OrderCreatedEvent struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
}

// OrderCreatedV1 is the typed event definition
var OrderCreatedV1 = helper.EventDefinition[OrderCreatedEvent](
	"tracking",     // Module name (domain)
	"OrderCreated", // Event name
	"v1",           // Version
)

// ============================================================
// Tracking Module (Event Emitter)
// ============================================================

// TrackingModule emits order events
type TrackingModule struct {
	eventBus   mono.EventBus
	orderCount atomic.Int64
}

// Compile-time interface checks
var _ mono.EventEmitterModule = (*TrackingModule)(nil)

func NewTrackingModule() *TrackingModule {
	return &TrackingModule{}
}

func (m *TrackingModule) Name() string { return "tracking" }

func (m *TrackingModule) Start(_ context.Context) error {
	slog.Info("Tracking module started")
	return nil
}

func (m *TrackingModule) Stop(_ context.Context) error {
	slog.Info("Tracking module stopped")
	return nil
}

// SetEventBus receives the EventBus for publishing
func (m *TrackingModule) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents declares which events this module emits
func (m *TrackingModule) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		OrderCreatedV1.ToBase(),
	}
}

// CreateOrder creates an order and publishes an event
func (m *TrackingModule) CreateOrder(customerID string, amount float64, currency string) (string, error) {
	orderNum := m.orderCount.Add(1)
	orderID := fmt.Sprintf("ORD-%06d", orderNum)

	slog.Info("Creating order",
		"orderID", orderID,
		"customerID", customerID,
		"amount", amount)

	// Publish event (fire-and-forget)
	err := OrderCreatedV1.Publish(m.eventBus, OrderCreatedEvent{
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     amount,
		Currency:   currency,
		CreatedAt:  time.Now(),
	}, nil)

	if err != nil {
		slog.Error("Failed to publish event", "error", err)
		// Continue - event failure shouldn't fail order creation
	}

	return orderID, nil
}

// ============================================================
// Notification Module (Event Consumer)
// ============================================================

// NotificationModule consumes order events
type NotificationModule struct {
	notifications []OrderCreatedEvent
}

// Compile-time interface checks
var _ mono.EventConsumerModule = (*NotificationModule)(nil)

func NewNotificationModule() *NotificationModule {
	return &NotificationModule{
		notifications: make([]OrderCreatedEvent, 0),
	}
}

func (m *NotificationModule) Name() string { return "notification" }

func (m *NotificationModule) Start(_ context.Context) error {
	slog.Info("Notification module started")
	return nil
}

func (m *NotificationModule) Stop(_ context.Context) error {
	slog.Info("Notification module stopped")
	return nil
}

// RegisterEventConsumers registers event handlers
func (m *NotificationModule) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Discover event by name (no dependency on tracking module)
	eventDef, ok := registry.GetEventByName("OrderCreated", "v1", "tracking")
	if !ok {
		return fmt.Errorf("event OrderCreated.v1 not found from tracking module")
	}

	// Register consumer handler
	return registry.RegisterEventConsumer(
		eventDef,
		m.handleOrderCreated,
		m,
	)
}

func (m *NotificationModule) handleOrderCreated(
	ctx context.Context, msg *mono.Msg) error {
	var event OrderCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	slog.Info("Sending order notification",
		"orderID", event.OrderID,
		"customerID", event.CustomerID,
		"amount", event.Amount)

	// Store notification for demonstration
	m.notifications = append(m.notifications, event)

	return nil
}

// GetNotifications returns all received notifications
func (m *NotificationModule) GetNotifications() []OrderCreatedEvent {
	return m.notifications
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Event Emitter Example ===")

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create modules
	trackingModule := NewTrackingModule()
	notificationModule := NewNotificationModule()

	// Register modules
	// Note: Order doesn't matter - no dependencies between them
	if err := app.Register(trackingModule); err != nil {
		log.Fatalf("Failed to register tracking module: %v", err)
	}
	if err := app.Register(notificationModule); err != nil {
		log.Fatalf("Failed to register notification module: %v", err)
	}

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Printf("Registered modules: %v\n", app.Modules())

	// Wait for subscriptions to be ready
	time.Sleep(100 * time.Millisecond)

	// Test event communication
	fmt.Println("\n=== Testing Event Communication ===")

	// Create some orders (events will be published)
	trackingModule.CreateOrder("CUST-001", 99.99, "USD")
	trackingModule.CreateOrder("CUST-002", 149.99, "USD")
	trackingModule.CreateOrder("CUST-003", 29.99, "EUR")

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Show notifications received
	fmt.Println("\n=== Notifications Received ===")
	for _, n := range notificationModule.GetNotifications() {
		fmt.Printf("  Order %s - Customer %s - %.2f %s\n",
			n.OrderID, n.CustomerID, n.Amount, n.Currency)
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
