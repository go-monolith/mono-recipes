package chat

import "time"

// Room represents a chat room.
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a chat message.
type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// User represents a connected user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	RoomID   string `json:"room_id,omitempty"`
}
