package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/example/websocket-chat-demo/events"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ChatModule is the ServiceProviderModule for chat operations.
// It also implements EventEmitterModule to publish chat events.
type ChatModule struct {
	service  *Service
	eventBus mono.EventBus
}

// Compile-time interface checks.
var _ mono.Module = (*ChatModule)(nil)
var _ mono.ServiceProviderModule = (*ChatModule)(nil)
var _ mono.EventEmitterModule = (*ChatModule)(nil)
var _ mono.HealthCheckableModule = (*ChatModule)(nil)

// NewModule creates a new ChatModule.
func NewModule() *ChatModule {
	return &ChatModule{
		service: NewService(),
	}
}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// Start initializes the module.
func (m *ChatModule) Start(_ context.Context) error {
	log.Println("[chat] Module started")
	return nil
}

// Stop shuts down the module.
func (m *ChatModule) Stop(_ context.Context) error {
	log.Println("[chat] Module stopped")
	return nil
}

// Health returns the health status.
func (m *ChatModule) Health(_ context.Context) mono.HealthStatus {
	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
	}
}

// SetEventBus is called by the framework to inject the event bus.
func (m *ChatModule) SetEventBus(bus mono.EventBus) {
	m.eventBus = bus
}

// EmitEvents returns all event definitions this module can emit.
func (m *ChatModule) EmitEvents() []mono.BaseEventDefinition {
	return []mono.BaseEventDefinition{
		events.MessageSentV1.ToBase(),
		events.UserJoinedV1.ToBase(),
		events.UserLeftV1.ToBase(),
		events.RoomCreatedV1.ToBase(),
	}
}

// RegisterServices registers chat services for request-reply.
func (m *ChatModule) RegisterServices(container mono.ServiceContainer) error {
	// Create Room
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceCreateRoom,
		json.Unmarshal, json.Marshal,
		m.handleCreateRoom,
	); err != nil {
		return fmt.Errorf("failed to register create-room service: %w", err)
	}

	// List Rooms
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceListRooms,
		json.Unmarshal, json.Marshal,
		m.handleListRooms,
	); err != nil {
		return fmt.Errorf("failed to register list-rooms service: %w", err)
	}

	// Get Room
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceGetRoom,
		json.Unmarshal, json.Marshal,
		m.handleGetRoom,
	); err != nil {
		return fmt.Errorf("failed to register get-room service: %w", err)
	}

	// Join Room
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceJoinRoom,
		json.Unmarshal, json.Marshal,
		m.handleJoinRoom,
	); err != nil {
		return fmt.Errorf("failed to register join-room service: %w", err)
	}

	// Leave Room
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceLeaveRoom,
		json.Unmarshal, json.Marshal,
		m.handleLeaveRoom,
	); err != nil {
		return fmt.Errorf("failed to register leave-room service: %w", err)
	}

	// Send Message
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceSendMessage,
		json.Unmarshal, json.Marshal,
		m.handleSendMessage,
	); err != nil {
		return fmt.Errorf("failed to register send-message service: %w", err)
	}

	// Get History
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceGetHistory,
		json.Unmarshal, json.Marshal,
		m.handleGetHistory,
	); err != nil {
		return fmt.Errorf("failed to register get-history service: %w", err)
	}

	// Get Room Members
	if err := helper.RegisterTypedRequestReplyService(
		container, ServiceGetRoomMembers,
		json.Unmarshal, json.Marshal,
		m.handleGetRoomMembers,
	); err != nil {
		return fmt.Errorf("failed to register get-room-members service: %w", err)
	}

	log.Println("[chat] Registered services: create-room, list-rooms, get-room, join-room, leave-room, send-message, get-history, get-room-members")
	return nil
}

// Service handlers

func (m *ChatModule) handleCreateRoom(ctx context.Context, req CreateRoomRequest, _ *mono.Msg) (CreateRoomResponse, error) {
	room, err := m.service.CreateRoom(ctx, req.Name, req.CreatedBy)
	if err != nil {
		return CreateRoomResponse{}, err
	}

	// Emit RoomCreated event
	if m.eventBus != nil {
		event := events.RoomCreatedEvent{
			RoomID:    room.ID,
			RoomName:  room.Name,
			CreatedBy: req.CreatedBy,
			Timestamp: room.CreatedAt,
		}
		if err := events.RoomCreatedV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[chat] Warning: failed to publish RoomCreated event: %v", err)
		}
	}

	return CreateRoomResponse{Room: room}, nil
}

func (m *ChatModule) handleListRooms(ctx context.Context, _ ListRoomsRequest, _ *mono.Msg) (ListRoomsResponse, error) {
	rooms := m.service.ListRooms(ctx)
	return ListRoomsResponse{Rooms: rooms}, nil
}

func (m *ChatModule) handleGetRoom(ctx context.Context, req GetRoomRequest, _ *mono.Msg) (GetRoomResponse, error) {
	room, err := m.service.GetRoom(ctx, req.RoomID)
	if err != nil {
		return GetRoomResponse{}, err
	}
	return GetRoomResponse{Room: room}, nil
}

func (m *ChatModule) handleJoinRoom(ctx context.Context, req JoinRoomRequest, _ *mono.Msg) (JoinRoomResponse, error) {
	if err := m.service.JoinRoom(ctx, req.RoomID, req.UserID, req.Username); err != nil {
		return JoinRoomResponse{Success: false, Message: err.Error()}, nil
	}

	// Emit UserJoined event
	if m.eventBus != nil {
		event := events.UserJoinedEvent{
			RoomID:    req.RoomID,
			UserID:    req.UserID,
			Username:  req.Username,
			Timestamp: time.Now(),
		}
		if err := events.UserJoinedV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[chat] Warning: failed to publish UserJoined event: %v", err)
		}
	}

	return JoinRoomResponse{Success: true}, nil
}

func (m *ChatModule) handleLeaveRoom(ctx context.Context, req LeaveRoomRequest, _ *mono.Msg) (LeaveRoomResponse, error) {
	if err := m.service.LeaveRoom(ctx, req.RoomID, req.UserID); err != nil {
		return LeaveRoomResponse{Success: false}, nil
	}

	// Emit UserLeft event
	if m.eventBus != nil {
		event := events.UserLeftEvent{
			RoomID:    req.RoomID,
			UserID:    req.UserID,
			Username:  req.Username,
			Timestamp: time.Now(),
		}
		if err := events.UserLeftV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[chat] Warning: failed to publish UserLeft event: %v", err)
		}
	}

	return LeaveRoomResponse{Success: true}, nil
}

func (m *ChatModule) handleSendMessage(ctx context.Context, req SendMessageRequest, _ *mono.Msg) (SendMessageResponse, error) {
	msg, err := m.service.SendMessage(ctx, req.RoomID, req.UserID, req.Username, req.Content)
	if err != nil {
		return SendMessageResponse{}, err
	}

	// Emit MessageSent event
	if m.eventBus != nil {
		event := events.MessageSentEvent{
			MessageID: msg.ID,
			RoomID:    msg.RoomID,
			UserID:    msg.UserID,
			Username:  msg.Username,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		}
		if err := events.MessageSentV1.Publish(m.eventBus, event, nil); err != nil {
			log.Printf("[chat] Warning: failed to publish MessageSent event: %v", err)
		}
	}

	return SendMessageResponse{
		MessageID: msg.ID,
		Timestamp: msg.Timestamp,
	}, nil
}

func (m *ChatModule) handleGetHistory(ctx context.Context, req GetHistoryRequest, _ *mono.Msg) (GetHistoryResponse, error) {
	messages, err := m.service.GetHistory(ctx, req.RoomID, req.Limit)
	if err != nil {
		return GetHistoryResponse{}, err
	}
	return GetHistoryResponse{Messages: messages}, nil
}

func (m *ChatModule) handleGetRoomMembers(ctx context.Context, req GetRoomMembersRequest, _ *mono.Msg) (GetRoomMembersResponse, error) {
	members, err := m.service.GetRoomMembers(ctx, req.RoomID)
	if err != nil {
		return GetRoomMembersResponse{}, err
	}
	return GetRoomMembersResponse{Members: members}, nil
}

// GetService returns the underlying chat service (for adapter use).
func (m *ChatModule) GetService() *Service {
	return m.service
}
