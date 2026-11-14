package wsserver

import (
	"context"

	chatcontract "github.com/example/websocket-chat-demo/modules/chat/contract"
)

// ChatPort is a driven port: what the WebSocket and REST handlers need from
// the chat module. The ws-server module owns it, it speaks the chat
// contract's DTOs, and it says nothing about NATS. Domain errors reported by
// the chat services are returned as Go errors.
type ChatPort interface {
	CreateRoom(ctx context.Context, name string) (*chatcontract.Room, error)
	GetRoom(ctx context.Context, roomID string) (*chatcontract.Room, bool, error)
	ListRooms(ctx context.Context) ([]chatcontract.Room, error)
	JoinRoom(ctx context.Context, roomID, userID, username string) (*chatcontract.User, error)
	LeaveRoom(ctx context.Context, userID string) error
	SendMessage(ctx context.Context, userID, content string) error
	GetUser(ctx context.Context, userID string) (*chatcontract.User, bool, error)
	GetHistory(ctx context.Context, roomID string, limit int) ([]chatcontract.Message, error)
	GetRoomUsers(ctx context.Context, roomID string) ([]chatcontract.User, error)
}
