package chat

import (
	"time"

	domain "github.com/example/websocket-chat-demo/domain/chat"
)

// Service names for request-reply pattern.
const (
	ServiceCreateRoom     = "chat.create-room"
	ServiceListRooms      = "chat.list-rooms"
	ServiceGetRoom        = "chat.get-room"
	ServiceJoinRoom       = "chat.join-room"
	ServiceLeaveRoom      = "chat.leave-room"
	ServiceSendMessage    = "chat.send-message"
	ServiceGetHistory     = "chat.get-history"
	ServiceGetRoomMembers = "chat.get-room-members"
)

// CreateRoomRequest is the request to create a new room.
type CreateRoomRequest struct {
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

// CreateRoomResponse is the response after creating a room.
type CreateRoomResponse struct {
	Room *domain.Room `json:"room"`
}

// ListRoomsRequest is the request to list all rooms.
type ListRoomsRequest struct{}

// ListRoomsResponse is the response with all rooms.
type ListRoomsResponse struct {
	Rooms []*domain.Room `json:"rooms"`
}

// GetRoomRequest is the request to get a room by ID.
type GetRoomRequest struct {
	RoomID string `json:"room_id"`
}

// GetRoomResponse is the response with the room.
type GetRoomResponse struct {
	Room *domain.Room `json:"room"`
}

// JoinRoomRequest is the request to join a room.
type JoinRoomRequest struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// JoinRoomResponse is the response after joining a room.
type JoinRoomResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// LeaveRoomRequest is the request to leave a room.
type LeaveRoomRequest struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// LeaveRoomResponse is the response after leaving a room.
type LeaveRoomResponse struct {
	Success bool `json:"success"`
}

// SendMessageRequest is the request to send a message.
type SendMessageRequest struct {
	RoomID   string `json:"room_id"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Content  string `json:"content"`
}

// SendMessageResponse is the response after sending a message.
type SendMessageResponse struct {
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

// GetHistoryRequest is the request to get message history.
type GetHistoryRequest struct {
	RoomID string `json:"room_id"`
	Limit  int    `json:"limit,omitempty"`
}

// GetHistoryResponse is the response with message history.
type GetHistoryResponse struct {
	Messages []*domain.Message `json:"messages"`
}

// GetRoomMembersRequest is the request to get room members.
type GetRoomMembersRequest struct {
	RoomID string `json:"room_id"`
}

// GetRoomMembersResponse is the response with room members.
type GetRoomMembersResponse struct {
	Members []*domain.User `json:"members"`
}
