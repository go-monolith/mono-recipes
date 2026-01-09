package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"
	"github.com/google/uuid"
)

// Service names for Request-Reply services.
const (
	ServiceCreateRoom   = "create-room"
	ServiceGetRoom      = "get-room"
	ServiceListRooms    = "list-rooms"
	ServiceJoinRoom     = "join-room"
	ServiceLeaveRoom    = "leave-room"
	ServiceSendMessage  = "send-message"
	ServiceGetUser      = "get-user"
	ServiceGetHistory   = "get-history"
	ServiceGetRoomUsers = "get-room-users"
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

// RegisterServices registers Request-Reply services for inter-module communication.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	services := []struct {
		name    string
		handler mono.RequestReplyHandler
	}{
		{ServiceCreateRoom, m.handleCreateRoom},
		{ServiceGetRoom, m.handleGetRoom},
		{ServiceListRooms, m.handleListRooms},
		{ServiceJoinRoom, m.handleJoinRoom},
		{ServiceLeaveRoom, m.handleLeaveRoom},
		{ServiceSendMessage, m.handleSendMessage},
		{ServiceGetUser, m.handleGetUser},
		{ServiceGetHistory, m.handleGetHistory},
		{ServiceGetRoomUsers, m.handleGetRoomUsers},
	}

	for _, svc := range services {
		if err := container.RegisterRequestReplyService(svc.name, svc.handler); err != nil {
			return fmt.Errorf("failed to register service %s: %w", svc.name, err)
		}
	}

	m.logger.Info("Registered chat services", "count", len(services))
	return nil
}

// Service request/response types

// CreateRoomServiceRequest is the request for creating a room via service.
type CreateRoomServiceRequest struct {
	Name string `json:"name"`
}

// CreateRoomServiceResponse is the response for creating a room via service.
type CreateRoomServiceResponse struct {
	Room  *Room  `json:"room,omitempty"`
	Error string `json:"error,omitempty"`
}

// GetRoomServiceRequest is the request for getting a room via service.
type GetRoomServiceRequest struct {
	RoomID string `json:"room_id"`
}

// GetRoomServiceResponse is the response for getting a room via service.
type GetRoomServiceResponse struct {
	Room   *Room  `json:"room,omitempty"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

// ListRoomsServiceResponse is the response for listing rooms via service.
type ListRoomsServiceResponse struct {
	Rooms []Room `json:"rooms"`
}

// JoinRoomServiceRequest is the request for joining a room via service.
type JoinRoomServiceRequest struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// JoinRoomServiceResponse is the response for joining a room via service.
type JoinRoomServiceResponse struct {
	User  *User  `json:"user,omitempty"`
	Error string `json:"error,omitempty"`
}

// LeaveRoomServiceRequest is the request for leaving a room via service.
type LeaveRoomServiceRequest struct {
	UserID string `json:"user_id"`
}

// LeaveRoomServiceResponse is the response for leaving a room via service.
type LeaveRoomServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SendMessageServiceRequest is the request for sending a message via service.
type SendMessageServiceRequest struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
}

// SendMessageServiceResponse is the response for sending a message via service.
type SendMessageServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// GetUserServiceRequest is the request for getting a user via service.
type GetUserServiceRequest struct {
	UserID string `json:"user_id"`
}

// GetUserServiceResponse is the response for getting a user via service.
type GetUserServiceResponse struct {
	User   *User  `json:"user,omitempty"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

// GetHistoryServiceRequest is the request for getting message history via service.
type GetHistoryServiceRequest struct {
	RoomID string `json:"room_id"`
	Limit  int    `json:"limit"`
}

// GetHistoryServiceResponse is the response for getting message history via service.
type GetHistoryServiceResponse struct {
	Messages []Message `json:"messages"`
	Error    string    `json:"error,omitempty"`
}

// GetRoomUsersServiceRequest is the request for getting room users via service.
type GetRoomUsersServiceRequest struct {
	RoomID string `json:"room_id"`
}

// GetRoomUsersServiceResponse is the response for getting room users via service.
type GetRoomUsersServiceResponse struct {
	Users []User `json:"users"`
	Error string `json:"error,omitempty"`
}

// Service handlers

func (m *Module) handleCreateRoom(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req CreateRoomServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(CreateRoomServiceResponse{Error: "invalid request"})
	}

	room, err := m.CreateRoom(req.Name)
	if err != nil {
		return json.Marshal(CreateRoomServiceResponse{Error: err.Error()})
	}
	return json.Marshal(CreateRoomServiceResponse{Room: room})
}

func (m *Module) handleGetRoom(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req GetRoomServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(GetRoomServiceResponse{Error: "invalid request"})
	}

	room, exists := m.GetRoom(req.RoomID)
	return json.Marshal(GetRoomServiceResponse{Room: room, Exists: exists})
}

func (m *Module) handleListRooms(_ context.Context, _ *mono.Msg) ([]byte, error) {
	rooms := m.ListRooms()
	return json.Marshal(ListRoomsServiceResponse{Rooms: rooms})
}

func (m *Module) handleJoinRoom(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req JoinRoomServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(JoinRoomServiceResponse{Error: "invalid request"})
	}

	user, err := m.JoinRoom(req.RoomID, req.UserID, req.Username)
	if err != nil {
		return json.Marshal(JoinRoomServiceResponse{Error: err.Error()})
	}
	return json.Marshal(JoinRoomServiceResponse{User: user})
}

func (m *Module) handleLeaveRoom(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req LeaveRoomServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(LeaveRoomServiceResponse{Error: "invalid request"})
	}

	m.LeaveRoom(req.UserID)
	return json.Marshal(LeaveRoomServiceResponse{Success: true})
}

func (m *Module) handleSendMessage(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req SendMessageServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(SendMessageServiceResponse{Error: "invalid request"})
	}

	if err := m.SendMessage(req.UserID, req.Content); err != nil {
		return json.Marshal(SendMessageServiceResponse{Error: err.Error()})
	}
	return json.Marshal(SendMessageServiceResponse{Success: true})
}

func (m *Module) handleGetUser(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req GetUserServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(GetUserServiceResponse{Error: "invalid request"})
	}

	user, exists := m.store.GetUser(req.UserID)
	return json.Marshal(GetUserServiceResponse{User: user, Exists: exists})
}

func (m *Module) handleGetHistory(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req GetHistoryServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(GetHistoryServiceResponse{Error: "invalid request"})
	}

	messages := m.GetHistory(req.RoomID, req.Limit)
	return json.Marshal(GetHistoryServiceResponse{Messages: messages})
}

func (m *Module) handleGetRoomUsers(_ context.Context, msg *mono.Msg) ([]byte, error) {
	var req GetRoomUsersServiceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return json.Marshal(GetRoomUsersServiceResponse{Error: "invalid request"})
	}

	users := m.GetRoomUsers(req.RoomID)
	return json.Marshal(GetRoomUsersServiceResponse{Users: users})
}
