package events

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// MessageSentEvent is emitted when a user sends a message.
type MessageSentEvent struct {
	MessageID string    `json:"message_id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// UserJoinedEvent is emitted when a user joins a room.
type UserJoinedEvent struct {
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// UserLeftEvent is emitted when a user leaves a room.
type UserLeftEvent struct {
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// RoomCreatedEvent is emitted when a new room is created.
type RoomCreatedEvent struct {
	RoomID    string    `json:"room_id"`
	RoomName  string    `json:"room_name"`
	CreatedBy string    `json:"created_by"`
	Timestamp time.Time `json:"timestamp"`
}

// Event definitions for the chat domain.
var (
	MessageSentV1 = helper.EventDefinition[MessageSentEvent](
		"chat",
		"MessageSent",
		"v1",
	)

	UserJoinedV1 = helper.EventDefinition[UserJoinedEvent](
		"chat",
		"UserJoined",
		"v1",
	)

	UserLeftV1 = helper.EventDefinition[UserLeftEvent](
		"chat",
		"UserLeft",
		"v1",
	)

	RoomCreatedV1 = helper.EventDefinition[RoomCreatedEvent](
		"chat",
		"RoomCreated",
		"v1",
	)
)
