package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/example/websocket-chat-demo/domain/chat"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ChatPort defines the interface for chat operations.
type ChatPort interface {
	CreateRoom(ctx context.Context, name, createdBy string) (*domain.Room, error)
	ListRooms(ctx context.Context) ([]*domain.Room, error)
	GetRoom(ctx context.Context, roomID string) (*domain.Room, error)
	JoinRoom(ctx context.Context, roomID, userID, username string) error
	LeaveRoom(ctx context.Context, roomID, userID, username string) error
	SendMessage(ctx context.Context, roomID, userID, username, content string) (string, time.Time, error)
	GetHistory(ctx context.Context, roomID string, limit int) ([]*domain.Message, error)
	GetRoomMembers(ctx context.Context, roomID string) ([]*domain.User, error)
}

// ChatAdapter implements ChatPort using the service container.
type ChatAdapter struct {
	container mono.ServiceContainer
}

// NewChatAdapter creates a new ChatAdapter.
func NewChatAdapter(container mono.ServiceContainer) ChatPort {
	if container == nil {
		panic("chat: ServiceContainer is nil")
	}
	return &ChatAdapter{container: container}
}

// CreateRoom creates a new chat room.
func (a *ChatAdapter) CreateRoom(ctx context.Context, name, createdBy string) (*domain.Room, error) {
	req := CreateRoomRequest{Name: name, CreatedBy: createdBy}
	var resp CreateRoomResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceCreateRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	return resp.Room, nil
}

// ListRooms returns all available rooms.
func (a *ChatAdapter) ListRooms(ctx context.Context) ([]*domain.Room, error) {
	req := ListRoomsRequest{}
	var resp ListRoomsResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceListRooms,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}
	return resp.Rooms, nil
}

// GetRoom retrieves a room by ID.
func (a *ChatAdapter) GetRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	req := GetRoomRequest{RoomID: roomID}
	var resp GetRoomResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return resp.Room, nil
}

// JoinRoom adds a user to a room.
func (a *ChatAdapter) JoinRoom(ctx context.Context, roomID, userID, username string) error {
	req := JoinRoomRequest{RoomID: roomID, UserID: userID, Username: username}
	var resp JoinRoomResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceJoinRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("failed to join room: %s", resp.Message)
	}
	return nil
}

// LeaveRoom removes a user from a room.
func (a *ChatAdapter) LeaveRoom(ctx context.Context, roomID, userID, username string) error {
	req := LeaveRoomRequest{RoomID: roomID, UserID: userID, Username: username}
	var resp LeaveRoomResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceLeaveRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("failed to leave room: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("failed to leave room")
	}
	return nil
}

// SendMessage sends a message to a room.
func (a *ChatAdapter) SendMessage(ctx context.Context, roomID, userID, username, content string) (string, time.Time, error) {
	req := SendMessageRequest{RoomID: roomID, UserID: userID, Username: username, Content: content}
	var resp SendMessageResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceSendMessage,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to send message: %w", err)
	}
	return resp.MessageID, resp.Timestamp, nil
}

// GetHistory retrieves message history for a room.
func (a *ChatAdapter) GetHistory(ctx context.Context, roomID string, limit int) ([]*domain.Message, error) {
	req := GetHistoryRequest{RoomID: roomID, Limit: limit}
	var resp GetHistoryResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetHistory,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	return resp.Messages, nil
}

// GetRoomMembers returns all members in a room.
func (a *ChatAdapter) GetRoomMembers(ctx context.Context, roomID string) ([]*domain.User, error) {
	req := GetRoomMembersRequest{RoomID: roomID}
	var resp GetRoomMembersResponse
	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetRoomMembers,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to get room members: %w", err)
	}
	return resp.Members, nil
}

// WSMessage represents a WebSocket message format.
type WSMessage struct {
	Type      string          `json:"type"`
	RoomID    string          `json:"room_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	Username  string          `json:"username,omitempty"`
	Content   string          `json:"content,omitempty"`
	MessageID string          `json:"message_id,omitempty"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// Message types for WebSocket communication.
const (
	WSTypeJoin      = "join"
	WSTypeLeave     = "leave"
	WSTypeMessage   = "message"
	WSTypeHistory   = "history"
	WSTypeMembers   = "members"
	WSTypeError     = "error"
	WSTypeJoined    = "joined"
	WSTypeLeft      = "left"
	WSTypeBroadcast = "broadcast"
	WSTypeRoomList  = "room_list"
)
