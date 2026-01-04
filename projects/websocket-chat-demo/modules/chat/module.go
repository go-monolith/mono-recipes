package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/google/uuid"
)

// Module implements the chat room module with EventBus integration.
type Module struct {
	store    *RoomStore
	eventBus mono.EventBus
	logger   types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module              = (*Module)(nil)
	_ mono.EventBusAwareModule = (*Module)(nil)
	_ mono.EventEmitterModule  = (*Module)(nil)
	_ mono.EventConsumerModule = (*Module)(nil)
)

// NewModule creates a new chat module.
func NewModule(logger types.Logger) *Module {
	return &Module{
		store:  NewRoomStore(100),
		logger: logger,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "chat"
}

// SetEventBus receives the EventBus from the framework.
func (m *Module) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents declares the events this module can emit.
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		ChatMessageV1.ToBase(),
		UserJoinedV1.ToBase(),
		UserLeftV1.ToBase(),
	}
}

// RegisterEventConsumers registers event handlers for chat events.
// The chat module consumes its own events to store messages in history.
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
	// Register consumer for ChatMessage events to store in history
	msgDef, ok := registry.GetEventByName("ChatMessage", "v1", "chat")
	if !ok {
		return fmt.Errorf("event ChatMessage.v1 not found")
	}
	if err := registry.RegisterEventConsumer(msgDef, m.handleChatMessage, m); err != nil {
		return fmt.Errorf("failed to register ChatMessage consumer: %w", err)
	}

	m.logger.Info("Registered chat event consumers")
	return nil
}

// handleChatMessage stores messages in room history.
func (m *Module) handleChatMessage(_ context.Context, msg *mono.Msg) error {
	var event ChatEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		m.logger.Error("Failed to unmarshal ChatMessage event", "error", err)
		return nil // Don't retry on unmarshal errors
	}

	// Store message in history
	m.store.AddMessage(event.Message)
	m.logger.Debug("Stored message in history",
		"roomID", event.RoomID,
		"messageID", event.Message.ID)

	return nil
}

// Start initializes the chat module with a default lobby room.
func (m *Module) Start(ctx context.Context) error {
	// Create default lobby room
	m.store.CreateRoom("lobby", "General Lobby")
	m.logger.Info("Chat module started with default lobby room")
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("Chat module stopped")
	return nil
}

// Store returns the room store.
func (m *Module) Store() *RoomStore {
	return m.store
}

// CreateRoom creates a new chat room.
func (m *Module) CreateRoom(name string) (*Room, error) {
	if err := ValidateRoomName(name); err != nil {
		return nil, err
	}
	id := uuid.New().String()[:8]
	return m.store.CreateRoom(id, name), nil
}

// GetRoom returns a room by ID.
func (m *Module) GetRoom(roomID string) (*Room, bool) {
	return m.store.GetRoom(roomID)
}

// ListRooms returns all active rooms.
func (m *Module) ListRooms() []Room {
	return m.store.ListRooms()
}

// JoinRoom adds a user to a room and publishes a join event.
func (m *Module) JoinRoom(roomID, userID, username string) (*User, error) {
	// Validate username
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}

	user := &User{
		ID:       userID,
		Username: username,
		RoomID:   roomID,
	}

	if !m.store.JoinRoom(roomID, user) {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	// Publish join event
	event := ChatEvent{
		Type:   "join",
		RoomID: roomID,
		Message: Message{
			ID:        uuid.New().String(),
			RoomID:    roomID,
			UserID:    userID,
			Username:  username,
			Content:   fmt.Sprintf("%s joined the room", username),
			Timestamp: time.Now(),
			Type:      "join",
		},
	}

	if err := UserJoinedV1.Publish(m.eventBus, event, nil); err != nil {
		slog.Warn("Failed to publish UserJoined event", "error", err)
	}

	m.logger.Info("User joined room", "userID", userID, "roomID", roomID)
	return user, nil
}

// LeaveRoom removes a user from a room and publishes a leave event.
func (m *Module) LeaveRoom(userID string) {
	user, exists := m.store.GetUser(userID)
	if !exists {
		return
	}

	roomID := user.RoomID
	m.store.LeaveRoom(userID)

	// Publish leave event
	event := ChatEvent{
		Type:   "leave",
		RoomID: roomID,
		Message: Message{
			ID:        uuid.New().String(),
			RoomID:    roomID,
			UserID:    userID,
			Username:  user.Username,
			Content:   fmt.Sprintf("%s left the room", user.Username),
			Timestamp: time.Now(),
			Type:      "leave",
		},
	}

	if err := UserLeftV1.Publish(m.eventBus, event, nil); err != nil {
		slog.Warn("Failed to publish UserLeft event", "error", err)
	}

	m.logger.Info("User left room", "userID", userID, "roomID", roomID)
}

// SendMessage sends a message to a room and publishes a message event.
func (m *Module) SendMessage(userID, content string) error {
	// Validate message content
	if err := ValidateMessage(content); err != nil {
		return err
	}

	user, exists := m.store.GetUser(userID)
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	if user.RoomID == "" {
		return fmt.Errorf("user not in a room")
	}

	msg := Message{
		ID:        uuid.New().String(),
		RoomID:    user.RoomID,
		UserID:    userID,
		Username:  user.Username,
		Content:   content,
		Timestamp: time.Now(),
		Type:      "message",
	}

	// Publish message event
	event := ChatEvent{
		Type:    "message",
		RoomID:  user.RoomID,
		Message: msg,
	}

	if err := ChatMessageV1.Publish(m.eventBus, event, nil); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	m.logger.Debug("Message sent", "userID", userID, "roomID", user.RoomID)
	return nil
}

// GetHistory returns message history for a room.
func (m *Module) GetHistory(roomID string, limit int) []Message {
	return m.store.GetHistory(roomID, limit)
}

// GetRoomUsers returns all users in a room.
func (m *Module) GetRoomUsers(roomID string) []User {
	return m.store.GetRoomUsers(roomID)
}
