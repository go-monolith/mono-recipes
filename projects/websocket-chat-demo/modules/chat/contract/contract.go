// Package contract is the published port of the chat module: the DTOs of its
// services, the typed request-reply contracts other modules call them through,
// and the events the module emits. It is the ONLY package of the chat module
// another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// ModuleName is the name of the module that provides these contracts.
const ModuleName = "chat"

// ---------------------------------------------------------------------------
// Domain DTOs
// ---------------------------------------------------------------------------

// Message represents a chat message.
type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "join", "leave", "system"
}

// Room represents a chat room.
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UserCount int       `json:"user_count"`
}

// User represents a connected user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	RoomID   string `json:"room_id"`
}

// ---------------------------------------------------------------------------
// Service DTOs. Domain errors (validation, unknown room, ...) are reported in
// the Error field of the response, so callers can show them to the user; a
// Go error from Call means the service could not be reached.
// ---------------------------------------------------------------------------

// CreateRoomServiceRequest is the request for creating a room.
type CreateRoomServiceRequest struct {
	Name string `json:"name"`
}

// CreateRoomServiceResponse is the response for creating a room.
type CreateRoomServiceResponse struct {
	Room  *Room  `json:"room,omitempty"`
	Error string `json:"error,omitempty"`
}

// GetRoomServiceRequest is the request for getting a room.
type GetRoomServiceRequest struct {
	RoomID string `json:"room_id"`
}

// GetRoomServiceResponse is the response for getting a room.
type GetRoomServiceResponse struct {
	Room   *Room  `json:"room,omitempty"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

// ListRoomsServiceRequest is the (empty) request for listing rooms.
type ListRoomsServiceRequest struct{}

// ListRoomsServiceResponse is the response for listing rooms.
type ListRoomsServiceResponse struct {
	Rooms []Room `json:"rooms"`
}

// JoinRoomServiceRequest is the request for joining a room.
type JoinRoomServiceRequest struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// JoinRoomServiceResponse is the response for joining a room.
type JoinRoomServiceResponse struct {
	User  *User  `json:"user,omitempty"`
	Error string `json:"error,omitempty"`
}

// LeaveRoomServiceRequest is the request for leaving a room.
type LeaveRoomServiceRequest struct {
	UserID string `json:"user_id"`
}

// LeaveRoomServiceResponse is the response for leaving a room.
type LeaveRoomServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SendMessageServiceRequest is the request for sending a message.
type SendMessageServiceRequest struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
}

// SendMessageServiceResponse is the response for sending a message.
type SendMessageServiceResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// GetUserServiceRequest is the request for getting a user.
type GetUserServiceRequest struct {
	UserID string `json:"user_id"`
}

// GetUserServiceResponse is the response for getting a user.
type GetUserServiceResponse struct {
	User   *User  `json:"user,omitempty"`
	Exists bool   `json:"exists"`
	Error  string `json:"error,omitempty"`
}

// GetHistoryServiceRequest is the request for getting message history.
type GetHistoryServiceRequest struct {
	RoomID string `json:"room_id"`
	Limit  int    `json:"limit"`
}

// GetHistoryServiceResponse is the response for getting message history.
type GetHistoryServiceResponse struct {
	Messages []Message `json:"messages"`
	Error    string    `json:"error,omitempty"`
}

// GetRoomUsersServiceRequest is the request for getting room users.
type GetRoomUsersServiceRequest struct {
	RoomID string `json:"room_id"`
}

// GetRoomUsersServiceResponse is the response for getting room users.
type GetRoomUsersServiceResponse struct {
	Users []User `json:"users"`
	Error string `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Service contracts (subjects services.chat.<service>)
// ---------------------------------------------------------------------------

var (
	// CreateRoom creates a chat room.
	CreateRoom = helper.RequestReplyService[CreateRoomServiceRequest, CreateRoomServiceResponse](ModuleName, "create-room")
	// GetRoom returns a room by ID.
	GetRoom = helper.RequestReplyService[GetRoomServiceRequest, GetRoomServiceResponse](ModuleName, "get-room")
	// ListRooms returns all active rooms.
	ListRooms = helper.RequestReplyService[ListRoomsServiceRequest, ListRoomsServiceResponse](ModuleName, "list-rooms")
	// JoinRoom adds a user to a room and emits UserJoinedV1.
	JoinRoom = helper.RequestReplyService[JoinRoomServiceRequest, JoinRoomServiceResponse](ModuleName, "join-room")
	// LeaveRoom removes a user from their room and emits UserLeftV1.
	LeaveRoom = helper.RequestReplyService[LeaveRoomServiceRequest, LeaveRoomServiceResponse](ModuleName, "leave-room")
	// SendMessage sends a message to the user's room and emits ChatMessageV1.
	SendMessage = helper.RequestReplyService[SendMessageServiceRequest, SendMessageServiceResponse](ModuleName, "send-message")
	// GetUser returns a user by ID.
	GetUser = helper.RequestReplyService[GetUserServiceRequest, GetUserServiceResponse](ModuleName, "get-user")
	// GetHistory returns the message history of a room.
	GetHistory = helper.RequestReplyService[GetHistoryServiceRequest, GetHistoryServiceResponse](ModuleName, "get-history")
	// GetRoomUsers returns the users in a room.
	GetRoomUsers = helper.RequestReplyService[GetRoomUsersServiceRequest, GetRoomUsersServiceResponse](ModuleName, "get-room-users")
)

// ---------------------------------------------------------------------------
// Events emitted by the chat module
// ---------------------------------------------------------------------------

// ChatEvent is published when a chat event occurs.
type ChatEvent struct {
	Type    string  `json:"type"` // "message", "join", "leave"
	RoomID  string  `json:"room_id"`
	Message Message `json:"message"`
}

var (
	// ChatMessageV1 is published when a user sends a message.
	ChatMessageV1 = helper.EventDefinition[ChatEvent](ModuleName, "ChatMessage", "v1")

	// UserJoinedV1 is published when a user joins a room.
	UserJoinedV1 = helper.EventDefinition[ChatEvent](ModuleName, "UserJoined", "v1")

	// UserLeftV1 is published when a user leaves a room.
	UserLeftV1 = helper.EventDefinition[ChatEvent](ModuleName, "UserLeft", "v1")
)
