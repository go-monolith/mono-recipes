// Package chat holds the domain logic and the driving adapter of the chat
// module. Other modules import chat/contract, not this package.
package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/google/uuid"

	"github.com/example/websocket-chat-demo/modules/chat/contract"
)

// Module implements the chat room module with EventBus integration.
type Module struct {
	store    *RoomStore
	eventBus mono.EventBus
	logger   types.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.EventBusAwareModule   = (*Module)(nil)
	_ mono.EventEmitterModule    = (*Module)(nil)
	_ mono.EventConsumerModule   = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
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
	return contract.ModuleName
}

// SetEventBus receives the EventBus from the framework.
func (m *Module) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents declares the events this module can emit. The definitions live
// in the contract package so consumers (ws-server) can subscribe without
// importing this package.
func (m *Module) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		contract.ChatMessageV1.ToBase(),
		contract.UserJoinedV1.ToBase(),
		contract.UserLeftV1.ToBase(),
	}
}

// RegisterEventConsumers registers event handlers for chat events.
// The chat module consumes its own ChatMessage events to store messages in
// history.
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
	if err := helper.RegisterTypedEventConsumer(registry, contract.ChatMessageV1, m.handleChatMessage, m); err != nil {
		return fmt.Errorf("failed to register ChatMessage consumer: %w", err)
	}

	m.logger.Info("Registered chat event consumers")
	return nil
}

// handleChatMessage stores messages in room history.
func (m *Module) handleChatMessage(_ context.Context, event contract.ChatEvent, _ *mono.Msg) error {
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
	event := contract.ChatEvent{
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

	if err := contract.UserJoinedV1.Publish(m.eventBus, event, nil); err != nil {
		m.logger.Warn("Failed to publish UserJoined event", "error", err)
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
	event := contract.ChatEvent{
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

	if err := contract.UserLeftV1.Publish(m.eventBus, event, nil); err != nil {
		m.logger.Warn("Failed to publish UserLeft event", "error", err)
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
	event := contract.ChatEvent{
		Type:    "message",
		RoomID:  user.RoomID,
		Message: msg,
	}

	if err := contract.ChatMessageV1.Publish(m.eventBus, event, nil); err != nil {
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

// RegisterServices is the driving adapter: it binds the typed handlers to the
// module's own contracts. No service-name strings, no manual JSON.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	if err := errors.Join(
		contract.CreateRoom.Register(container, m.handleCreateRoom),
		contract.GetRoom.Register(container, m.handleGetRoom),
		contract.ListRooms.Register(container, m.handleListRooms),
		contract.JoinRoom.Register(container, m.handleJoinRoom),
		contract.LeaveRoom.Register(container, m.handleLeaveRoom),
		contract.SendMessage.Register(container, m.handleSendMessage),
		contract.GetUser.Register(container, m.handleGetUser),
		contract.GetHistory.Register(container, m.handleGetHistory),
		contract.GetRoomUsers.Register(container, m.handleGetRoomUsers),
	); err != nil {
		return fmt.Errorf("failed to register chat services: %w", err)
	}

	m.logger.Info("Registered chat services")
	return nil
}

// Service handlers. Request decoding and response encoding are done by the
// contracts; domain errors go into the response's Error field.

func (m *Module) handleCreateRoom(_ context.Context, req contract.CreateRoomServiceRequest, _ *mono.Msg) (contract.CreateRoomServiceResponse, error) {
	room, err := m.CreateRoom(req.Name)
	if err != nil {
		return contract.CreateRoomServiceResponse{Error: err.Error()}, nil
	}
	return contract.CreateRoomServiceResponse{Room: room}, nil
}

func (m *Module) handleGetRoom(_ context.Context, req contract.GetRoomServiceRequest, _ *mono.Msg) (contract.GetRoomServiceResponse, error) {
	room, exists := m.GetRoom(req.RoomID)
	return contract.GetRoomServiceResponse{Room: room, Exists: exists}, nil
}

func (m *Module) handleListRooms(_ context.Context, _ contract.ListRoomsServiceRequest, _ *mono.Msg) (contract.ListRoomsServiceResponse, error) {
	return contract.ListRoomsServiceResponse{Rooms: m.ListRooms()}, nil
}

func (m *Module) handleJoinRoom(_ context.Context, req contract.JoinRoomServiceRequest, _ *mono.Msg) (contract.JoinRoomServiceResponse, error) {
	user, err := m.JoinRoom(req.RoomID, req.UserID, req.Username)
	if err != nil {
		return contract.JoinRoomServiceResponse{Error: err.Error()}, nil
	}
	return contract.JoinRoomServiceResponse{User: user}, nil
}

func (m *Module) handleLeaveRoom(_ context.Context, req contract.LeaveRoomServiceRequest, _ *mono.Msg) (contract.LeaveRoomServiceResponse, error) {
	m.LeaveRoom(req.UserID)
	return contract.LeaveRoomServiceResponse{Success: true}, nil
}

func (m *Module) handleSendMessage(_ context.Context, req contract.SendMessageServiceRequest, _ *mono.Msg) (contract.SendMessageServiceResponse, error) {
	if err := m.SendMessage(req.UserID, req.Content); err != nil {
		return contract.SendMessageServiceResponse{Error: err.Error()}, nil
	}
	return contract.SendMessageServiceResponse{Success: true}, nil
}

func (m *Module) handleGetUser(_ context.Context, req contract.GetUserServiceRequest, _ *mono.Msg) (contract.GetUserServiceResponse, error) {
	user, exists := m.store.GetUser(req.UserID)
	return contract.GetUserServiceResponse{User: user, Exists: exists}, nil
}

func (m *Module) handleGetHistory(_ context.Context, req contract.GetHistoryServiceRequest, _ *mono.Msg) (contract.GetHistoryServiceResponse, error) {
	return contract.GetHistoryServiceResponse{Messages: m.GetHistory(req.RoomID, req.Limit)}, nil
}

func (m *Module) handleGetRoomUsers(_ context.Context, req contract.GetRoomUsersServiceRequest, _ *mono.Msg) (contract.GetRoomUsersServiceResponse, error) {
	return contract.GetRoomUsersServiceResponse{Users: m.GetRoomUsers(req.RoomID)}, nil
}
