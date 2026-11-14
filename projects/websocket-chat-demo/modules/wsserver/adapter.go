package wsserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-monolith/mono"

	chatcontract "github.com/example/websocket-chat-demo/modules/chat/contract"
)

// chatAdapter is the driven adapter behind ChatPort: it calls the chat
// contracts with the container received in SetDependencyServiceContainer and
// turns the Error field of a response into a Go error. It is the only place
// in the module that knows the call crosses a module boundary.
type chatAdapter struct{ container mono.ServiceContainer }

var _ ChatPort = chatAdapter{}

// domainError returns the error reported in a response's Error field, if any.
func domainError(msg string) error {
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

func (a chatAdapter) CreateRoom(ctx context.Context, name string) (*chatcontract.Room, error) {
	resp, err := chatcontract.CreateRoom.Call(ctx, a.container, chatcontract.CreateRoomServiceRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to call create-room service: %w", err)
	}
	return resp.Room, domainError(resp.Error)
}

func (a chatAdapter) GetRoom(ctx context.Context, roomID string) (*chatcontract.Room, bool, error) {
	resp, err := chatcontract.GetRoom.Call(ctx, a.container, chatcontract.GetRoomServiceRequest{RoomID: roomID})
	if err != nil {
		return nil, false, fmt.Errorf("failed to call get-room service: %w", err)
	}
	return resp.Room, resp.Exists, domainError(resp.Error)
}

func (a chatAdapter) ListRooms(ctx context.Context) ([]chatcontract.Room, error) {
	resp, err := chatcontract.ListRooms.Call(ctx, a.container, chatcontract.ListRoomsServiceRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to call list-rooms service: %w", err)
	}
	return resp.Rooms, nil
}

func (a chatAdapter) JoinRoom(ctx context.Context, roomID, userID, username string) (*chatcontract.User, error) {
	resp, err := chatcontract.JoinRoom.Call(ctx, a.container, chatcontract.JoinRoomServiceRequest{
		RoomID: roomID, UserID: userID, Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call join-room service: %w", err)
	}
	return resp.User, domainError(resp.Error)
}

func (a chatAdapter) LeaveRoom(ctx context.Context, userID string) error {
	resp, err := chatcontract.LeaveRoom.Call(ctx, a.container, chatcontract.LeaveRoomServiceRequest{UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to call leave-room service: %w", err)
	}
	return domainError(resp.Error)
}

func (a chatAdapter) SendMessage(ctx context.Context, userID, content string) error {
	resp, err := chatcontract.SendMessage.Call(ctx, a.container, chatcontract.SendMessageServiceRequest{
		UserID: userID, Content: content,
	})
	if err != nil {
		return fmt.Errorf("failed to call send-message service: %w", err)
	}
	return domainError(resp.Error)
}

func (a chatAdapter) GetUser(ctx context.Context, userID string) (*chatcontract.User, bool, error) {
	resp, err := chatcontract.GetUser.Call(ctx, a.container, chatcontract.GetUserServiceRequest{UserID: userID})
	if err != nil {
		return nil, false, fmt.Errorf("failed to call get-user service: %w", err)
	}
	return resp.User, resp.Exists, domainError(resp.Error)
}

func (a chatAdapter) GetHistory(ctx context.Context, roomID string, limit int) ([]chatcontract.Message, error) {
	resp, err := chatcontract.GetHistory.Call(ctx, a.container, chatcontract.GetHistoryServiceRequest{
		RoomID: roomID, Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call get-history service: %w", err)
	}
	return resp.Messages, domainError(resp.Error)
}

func (a chatAdapter) GetRoomUsers(ctx context.Context, roomID string) ([]chatcontract.User, error) {
	resp, err := chatcontract.GetRoomUsers.Call(ctx, a.container, chatcontract.GetRoomUsersServiceRequest{RoomID: roomID})
	if err != nil {
		return nil, fmt.Errorf("failed to call get-room-users service: %w", err)
	}
	return resp.Users, domainError(resp.Error)
}
