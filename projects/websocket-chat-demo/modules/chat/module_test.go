package chat_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/websocket-chat-demo/modules/chat"
	"github.com/example/websocket-chat-demo/modules/chat/contract"
)

// nopLogger is the module logger in tests (the harness logs framework events).
type nopLogger struct{}

func (nopLogger) Debug(string, ...any)             {}
func (nopLogger) Info(string, ...any)              {}
func (nopLogger) Warn(string, ...any)              {}
func (nopLogger) Error(string, ...any)             {}
func (l nopLogger) With(...any) types.Logger       { return l }
func (l nopLogger) WithModule(string) types.Logger { return l }
func (l nopLogger) WithError(error) types.Logger   { return l }

// exerciseChat calls every chat contract through h.Container(), the module's
// own container, and checks the round trip of a message through the
// ChatMessage event the module consumes itself.
func exerciseChat(ctx context.Context, h *monotest.Harness) error {
	c := h.Container()

	created, err := contract.CreateRoom.Call(ctx, c, contract.CreateRoomServiceRequest{Name: "Go"})
	if err != nil || created.Error != "" || created.Room == nil {
		return fmt.Errorf("create-room: %+v %v", created, err)
	}
	roomID := created.Room.ID

	if bad, err := contract.CreateRoom.Call(ctx, c, contract.CreateRoomServiceRequest{}); err != nil || bad.Error == "" {
		return fmt.Errorf("create-room with an empty name should report an error: %+v %v", bad, err)
	}

	room, err := contract.GetRoom.Call(ctx, c, contract.GetRoomServiceRequest{RoomID: roomID})
	if err != nil || !room.Exists {
		return fmt.Errorf("get-room: %+v %v", room, err)
	}

	rooms, err := contract.ListRooms.Call(ctx, c, contract.ListRoomsServiceRequest{})
	if err != nil || len(rooms.Rooms) != 2 { // lobby + Go
		return fmt.Errorf("list-rooms: %+v %v", rooms, err)
	}

	joined, err := contract.JoinRoom.Call(ctx, c, contract.JoinRoomServiceRequest{RoomID: roomID, UserID: "u1", Username: "alice"})
	if err != nil || joined.Error != "" {
		return fmt.Errorf("join-room: %+v %v", joined, err)
	}

	user, err := contract.GetUser.Call(ctx, c, contract.GetUserServiceRequest{UserID: "u1"})
	if err != nil || !user.Exists || user.User.RoomID != roomID {
		return fmt.Errorf("get-user: %+v %v", user, err)
	}

	users, err := contract.GetRoomUsers.Call(ctx, c, contract.GetRoomUsersServiceRequest{RoomID: roomID})
	if err != nil || len(users.Users) != 1 {
		return fmt.Errorf("get-room-users: %+v %v", users, err)
	}

	sent, err := contract.SendMessage.Call(ctx, c, contract.SendMessageServiceRequest{UserID: "u1", Content: "hello"})
	if err != nil || sent.Error != "" {
		return fmt.Errorf("send-message: %+v %v", sent, err)
	}

	// The message reaches the history through the ChatMessage event.
	deadline := time.Now().Add(5 * time.Second)
	for {
		history, err := contract.GetHistory.Call(ctx, c, contract.GetHistoryServiceRequest{RoomID: roomID, Limit: 10})
		if err != nil {
			return fmt.Errorf("get-history: %v", err)
		}
		if len(history.Messages) == 1 && history.Messages[0].Content == "hello" {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("message not stored in history: %+v", history)
		}
		time.Sleep(10 * time.Millisecond)
	}

	left, err := contract.LeaveRoom.Call(ctx, c, contract.LeaveRoomServiceRequest{UserID: "u1"})
	if err != nil || !left.Success {
		return fmt.Errorf("leave-room: %+v %v", left, err)
	}
	return nil
}

// TestChatModuleConformance is the module's boundary check: valid service
// subjects, every published event declared in EmitEvents, Start/Stop within
// the timeout, no boundary violation while every provided contract is called.
func TestChatModuleConformance(t *testing.T) {
	monotest.CheckModule(t, chat.NewModule(nopLogger{}),
		monotest.WithExercise(exerciseChat))
}

// TestChatEventsPublished checks the module publishes the three events of its
// contract while being driven through its services.
func TestChatEventsPublished(t *testing.T) {
	h := monotest.New(t, chat.NewModule(nopLogger{}))
	ctx := context.Background()
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	if err := exerciseChat(ctx, h); err != nil {
		t.Fatal(err)
	}

	published := map[string]bool{}
	for _, subject := range h.ModulePublished() {
		published[subject] = true
	}
	for _, def := range []types.EventDefinition[contract.ChatEvent]{contract.UserJoinedV1, contract.ChatMessageV1, contract.UserLeftV1} {
		if !published[def.Subject] {
			t.Errorf("event %s (%s) not published; published: %v", def.Name, def.Subject, h.ModulePublished())
		}
	}
	if v := h.Report().Violations; len(v) != 0 {
		t.Errorf("boundary violations: %+v", v)
	}
}
