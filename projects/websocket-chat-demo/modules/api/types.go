package api

import "time"

// CreateRoomRequest is the API request to create a room.
type CreateRoomRequest struct {
	Name string `json:"name"`
}

// RoomResponse is the API response for a room.
type RoomResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Members   int       `json:"members,omitempty"`
}

// RoomListResponse is the API response for listing rooms.
type RoomListResponse struct {
	Rooms []RoomResponse `json:"rooms"`
}

// MessageResponse is the API response for a message.
type MessageResponse struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// HistoryResponse is the API response for message history.
type HistoryResponse struct {
	RoomID   string            `json:"room_id"`
	Messages []MessageResponse `json:"messages"`
}

// ErrorResponse is the API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HealthResponse is the API health check response.
type HealthResponse struct {
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}
