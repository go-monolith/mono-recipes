package chat

import "github.com/go-monolith/mono/pkg/helper"

// Event definitions for the chat module.
var (
	// ChatMessageV1 is published when a user sends a message.
	ChatMessageV1 = helper.EventDefinition[ChatEvent](
		"chat",
		"ChatMessage",
		"v1",
	)

	// UserJoinedV1 is published when a user joins a room.
	UserJoinedV1 = helper.EventDefinition[ChatEvent](
		"chat",
		"UserJoined",
		"v1",
	)

	// UserLeftV1 is published when a user leaves a room.
	UserLeftV1 = helper.EventDefinition[ChatEvent](
		"chat",
		"UserLeft",
		"v1",
	)
)
