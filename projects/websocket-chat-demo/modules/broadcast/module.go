package broadcast

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/example/websocket-chat-demo/events"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// BroadcastModule is an EventConsumerModule that broadcasts chat events to WebSocket clients.
type BroadcastModule struct {
	hub       *Hub
	cancelHub context.CancelFunc
}

// Compile-time interface checks.
var _ mono.Module = (*BroadcastModule)(nil)
var _ mono.EventConsumerModule = (*BroadcastModule)(nil)
var _ mono.HealthCheckableModule = (*BroadcastModule)(nil)

// NewModule creates a new BroadcastModule.
func NewModule() *BroadcastModule {
	return &BroadcastModule{
		hub: NewHub(),
	}
}

// Name returns the module name.
func (m *BroadcastModule) Name() string {
	return "broadcast"
}

// Start initializes the module and starts the hub.
func (m *BroadcastModule) Start(_ context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelHub = cancel
	go m.hub.Run(ctx)
	log.Println("[broadcast] Module started - WebSocket hub running")
	return nil
}

// Stop shuts down the module.
func (m *BroadcastModule) Stop(_ context.Context) error {
	clientCount := m.hub.ClientCount()
	if m.cancelHub != nil {
		m.cancelHub()
		m.hub.Wait() // Wait for hub to finish
	}
	log.Printf("[broadcast] Module stopped - %d clients were connected", clientCount)
	return nil
}

// Health returns the health status.
func (m *BroadcastModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"connected_clients": m.hub.ClientCount(),
		},
	}
}

// RegisterEventConsumers registers event handlers.
func (m *BroadcastModule) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Subscribe to MessageSent events
	if err := helper.RegisterTypedEventConsumer(
		registry, events.MessageSentV1, m.handleMessageSent, m,
	); err != nil {
		return fmt.Errorf("failed to register MessageSent consumer: %w", err)
	}

	// Subscribe to UserJoined events
	if err := helper.RegisterTypedEventConsumer(
		registry, events.UserJoinedV1, m.handleUserJoined, m,
	); err != nil {
		return fmt.Errorf("failed to register UserJoined consumer: %w", err)
	}

	// Subscribe to UserLeft events
	if err := helper.RegisterTypedEventConsumer(
		registry, events.UserLeftV1, m.handleUserLeft, m,
	); err != nil {
		return fmt.Errorf("failed to register UserLeft consumer: %w", err)
	}

	// Subscribe to RoomCreated events
	if err := helper.RegisterTypedEventConsumer(
		registry, events.RoomCreatedV1, m.handleRoomCreated, m,
	); err != nil {
		return fmt.Errorf("failed to register RoomCreated consumer: %w", err)
	}

	log.Println("[broadcast] Registered event consumers: MessageSent, UserJoined, UserLeft, RoomCreated")
	return nil
}

// Event handlers

func (m *BroadcastModule) handleMessageSent(_ context.Context, event events.MessageSentEvent, _ *mono.Msg) error {
	log.Printf("[broadcast] Broadcasting message from %s in room %s", event.Username, event.RoomID)

	m.hub.Broadcast(event.RoomID, "message", WSBroadcast{
		Type:      "message",
		RoomID:    event.RoomID,
		MessageID: event.MessageID,
		UserID:    event.UserID,
		Username:  event.Username,
		Content:   event.Content,
		Timestamp: event.Timestamp,
	})

	return nil
}

func (m *BroadcastModule) handleUserJoined(_ context.Context, event events.UserJoinedEvent, _ *mono.Msg) error {
	log.Printf("[broadcast] Broadcasting user joined: %s in room %s", event.Username, event.RoomID)

	m.hub.Broadcast(event.RoomID, "user_joined", WSBroadcast{
		Type:      "user_joined",
		RoomID:    event.RoomID,
		UserID:    event.UserID,
		Username:  event.Username,
		Timestamp: event.Timestamp,
	})

	return nil
}

func (m *BroadcastModule) handleUserLeft(_ context.Context, event events.UserLeftEvent, _ *mono.Msg) error {
	log.Printf("[broadcast] Broadcasting user left: %s from room %s", event.Username, event.RoomID)

	m.hub.Broadcast(event.RoomID, "user_left", WSBroadcast{
		Type:      "user_left",
		RoomID:    event.RoomID,
		UserID:    event.UserID,
		Username:  event.Username,
		Timestamp: event.Timestamp,
	})

	return nil
}

func (m *BroadcastModule) handleRoomCreated(_ context.Context, event events.RoomCreatedEvent, _ *mono.Msg) error {
	log.Printf("[broadcast] Broadcasting room created: %s", event.RoomName)

	// Broadcast to all connected clients (no room filter)
	m.hub.Broadcast("", "room_created", WSBroadcast{
		Type:      "room_created",
		RoomID:    event.RoomID,
		Content:   event.RoomName,
		Timestamp: event.Timestamp,
	})

	return nil
}

// GetHub returns the WebSocket hub for the API module to use.
func (m *BroadcastModule) GetHub() *Hub {
	return m.hub
}

// WSBroadcast is the structure sent to WebSocket clients.
type WSBroadcast struct {
	Type      string    `json:"type"`
	RoomID    string    `json:"room_id,omitempty"`
	MessageID string    `json:"message_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Content   string    `json:"content,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}
