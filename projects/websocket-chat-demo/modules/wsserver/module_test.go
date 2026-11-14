package wsserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	chatcontract "github.com/example/websocket-chat-demo/modules/chat/contract"
	"github.com/example/websocket-chat-demo/modules/wsserver"
)

// nopLogger is the module logger in tests.
type nopLogger struct{}

func (nopLogger) Debug(string, ...any)             {}
func (nopLogger) Info(string, ...any)              {}
func (nopLogger) Warn(string, ...any)              {}
func (nopLogger) Error(string, ...any)             {}
func (l nopLogger) With(...any) types.Logger       { return l }
func (l nopLogger) WithModule(string) types.Logger { return l }
func (l nopLogger) WithError(error) types.Logger   { return l }

// freeAddr returns a localhost address with a free TCP port.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

// standInChat is a NATS-free stand-in for the chat module: a single "lobby"
// room and an in-memory user map. Its services are registered through the
// chat contracts, and like the real module it publishes the chat events on the
// shared bus, so the ws-server event consumers are exercised too.
func standInChat(t *testing.T) *monotest.Container {
	t.Helper()
	bus := monotest.NewEventBus()
	c := monotest.NewContainer(chatcontract.ModuleName, bus)
	lobby := chatcontract.Room{ID: "lobby", Name: "General Lobby"}
	var mu sync.Mutex
	users := map[string]chatcontract.User{}

	publish := func(def types.EventDefinition[chatcontract.ChatEvent], msgType string, u chatcontract.User, content string) {
		_ = def.Publish(bus, chatcontract.ChatEvent{Type: msgType, RoomID: u.RoomID, Message: chatcontract.Message{
			ID: "m-" + msgType, RoomID: u.RoomID, UserID: u.ID, Username: u.Username, Content: content, Type: msgType, Timestamp: time.Now(),
		}}, nil)
	}

	errs := []error{
		chatcontract.CreateRoom.Register(c, func(_ context.Context, req chatcontract.CreateRoomServiceRequest, _ *types.Msg) (chatcontract.CreateRoomServiceResponse, error) {
			if req.Name == "" {
				return chatcontract.CreateRoomServiceResponse{Error: "room name cannot be empty"}, nil
			}
			return chatcontract.CreateRoomServiceResponse{Room: &chatcontract.Room{ID: "r1", Name: req.Name}}, nil
		}),
		chatcontract.GetRoom.Register(c, func(_ context.Context, req chatcontract.GetRoomServiceRequest, _ *types.Msg) (chatcontract.GetRoomServiceResponse, error) {
			if req.RoomID != lobby.ID {
				return chatcontract.GetRoomServiceResponse{}, nil
			}
			return chatcontract.GetRoomServiceResponse{Room: &lobby, Exists: true}, nil
		}),
		chatcontract.ListRooms.Register(c, func(context.Context, chatcontract.ListRoomsServiceRequest, *types.Msg) (chatcontract.ListRoomsServiceResponse, error) {
			return chatcontract.ListRoomsServiceResponse{Rooms: []chatcontract.Room{lobby}}, nil
		}),
		chatcontract.JoinRoom.Register(c, func(_ context.Context, req chatcontract.JoinRoomServiceRequest, _ *types.Msg) (chatcontract.JoinRoomServiceResponse, error) {
			if req.RoomID != lobby.ID {
				return chatcontract.JoinRoomServiceResponse{Error: "room not found: " + req.RoomID}, nil
			}
			u := chatcontract.User{ID: req.UserID, Username: req.Username, RoomID: req.RoomID}
			mu.Lock()
			users[u.ID] = u
			mu.Unlock()
			publish(chatcontract.UserJoinedV1, "join", u, u.Username+" joined the room")
			return chatcontract.JoinRoomServiceResponse{User: &u}, nil
		}),
		chatcontract.LeaveRoom.Register(c, func(_ context.Context, req chatcontract.LeaveRoomServiceRequest, _ *types.Msg) (chatcontract.LeaveRoomServiceResponse, error) {
			mu.Lock()
			u, ok := users[req.UserID]
			delete(users, req.UserID)
			mu.Unlock()
			if ok {
				publish(chatcontract.UserLeftV1, "leave", u, u.Username+" left the room")
			}
			return chatcontract.LeaveRoomServiceResponse{Success: true}, nil
		}),
		chatcontract.SendMessage.Register(c, func(_ context.Context, req chatcontract.SendMessageServiceRequest, _ *types.Msg) (chatcontract.SendMessageServiceResponse, error) {
			mu.Lock()
			u, ok := users[req.UserID]
			mu.Unlock()
			if !ok {
				return chatcontract.SendMessageServiceResponse{Error: "user not found: " + req.UserID}, nil
			}
			publish(chatcontract.ChatMessageV1, "message", u, req.Content)
			return chatcontract.SendMessageServiceResponse{Success: true}, nil
		}),
		chatcontract.GetUser.Register(c, func(_ context.Context, req chatcontract.GetUserServiceRequest, _ *types.Msg) (chatcontract.GetUserServiceResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			u, ok := users[req.UserID]
			if !ok {
				return chatcontract.GetUserServiceResponse{}, nil
			}
			return chatcontract.GetUserServiceResponse{User: &u, Exists: true}, nil
		}),
		chatcontract.GetHistory.Register(c, func(_ context.Context, req chatcontract.GetHistoryServiceRequest, _ *types.Msg) (chatcontract.GetHistoryServiceResponse, error) {
			return chatcontract.GetHistoryServiceResponse{Messages: []chatcontract.Message{{ID: "h1", RoomID: req.RoomID, Content: "earlier"}}}, nil
		}),
		chatcontract.GetRoomUsers.Register(c, func(_ context.Context, req chatcontract.GetRoomUsersServiceRequest, _ *types.Msg) (chatcontract.GetRoomUsersServiceResponse, error) {
			mu.Lock()
			defer mu.Unlock()
			var result []chatcontract.User
			for _, u := range users {
				if u.RoomID == req.RoomID {
					result = append(result, u)
				}
			}
			return chatcontract.GetRoomUsersServiceResponse{Users: result}, nil
		}),
	}
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// wsMessage mirrors the WebSocket envelope of the ws-server protocol.
type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// expect reads WebSocket messages until one of type want arrives.
func expect(conn *websocket.Conn, want string) (wsMessage, error) {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return msg, fmt.Errorf("waiting for %q: %w", want, err)
		}
		if msg.Type == "error" {
			return msg, fmt.Errorf("waiting for %q: server error %q", want, msg.Error)
		}
		if msg.Type == want {
			return msg, nil
		}
	}
}

// exerciseREST drives the REST handlers: list-rooms, create-room, get-room
// and get-history.
func exerciseREST(ctx context.Context, addr string) error {
	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/health", "", http.StatusOK},
		{http.MethodGet, "/api/v1/rooms", "", http.StatusOK},
		{http.MethodPost, "/api/v1/rooms", `{"name":"Go"}`, http.StatusCreated},
		{http.MethodPost, "/api/v1/rooms", `{"name":""}`, http.StatusBadRequest},
		{http.MethodGet, "/api/v1/rooms/lobby/history?limit=10", "", http.StatusOK},
		{http.MethodGet, "/api/v1/rooms/nope/history", "", http.StatusNotFound},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, tc := range cases {
		req, err := http.NewRequestWithContext(ctx, tc.method, "http://"+addr+tc.path, strings.NewReader(tc.body))
		if err != nil {
			return err
		}
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			return fmt.Errorf("%s %s: status %d, want %d (body %s)", tc.method, tc.path, resp.StatusCode, tc.want, body)
		}
	}
	return nil
}

// exerciseWebSocket drives the WebSocket protocol: join (join-room), users
// (get-user, get-room-users), message (send-message, then the ChatMessage
// event broadcast), history (get-history), rooms (list-rooms), leave
// (leave-room).
func exerciseWebSocket(addr string) error {
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	send := func(msgType string, payload any) error {
		msg := map[string]any{"type": msgType}
		if payload != nil {
			msg["payload"] = payload
		}
		return conn.WriteJSON(msg)
	}
	steps := []struct {
		msgType string
		payload any
		want    string
	}{
		{"join", map[string]string{"room_id": "lobby", "username": "alice"}, "joined"},
		{"users", nil, "users"},
		{"message", map[string]string{"content": "hello"}, "chat_message"},
		{"history", nil, "history"},
		{"rooms", nil, "rooms"},
		{"leave", nil, "left"},
	}
	for _, step := range steps {
		if err := send(step.msgType, step.payload); err != nil {
			return err
		}
		msg, err := expect(conn, step.want)
		if err != nil {
			return err
		}
		if step.want == "chat_message" && !strings.Contains(string(msg.Payload), "hello") {
			return fmt.Errorf("broadcast payload %s", msg.Payload)
		}
	}
	return nil
}

// TestWSServerModuleConformance is the module's boundary check: the module
// reaches the chat module only through its contracts, declares all of them in
// ConsumeServices, and subscribes to the chat events by their contract
// definitions.
func TestWSServerModuleConformance(t *testing.T) {
	chat := standInChat(t)
	addr := freeAddr(t)
	monotest.CheckModule(t, wsserver.NewModule(addr, nopLogger{}),
		monotest.WithDependency(chatcontract.ModuleName, chat),
		monotest.WithExercise(func(ctx context.Context, _ *monotest.Harness) error {
			if err := exerciseREST(ctx, addr); err != nil {
				return err
			}
			return exerciseWebSocket(addr)
		}))

	for _, def := range []types.BaseServiceDefinition{
		chatcontract.CreateRoom.ToBase(), chatcontract.GetRoom.ToBase(), chatcontract.ListRooms.ToBase(),
		chatcontract.JoinRoom.ToBase(), chatcontract.LeaveRoom.ToBase(), chatcontract.SendMessage.ToBase(),
		chatcontract.GetUser.ToBase(), chatcontract.GetHistory.ToBase(), chatcontract.GetRoomUsers.ToBase(),
	} {
		if chat.Calls(def.Name) == 0 {
			t.Errorf("chat contract %s was never called", def.Name)
		}
	}
}
