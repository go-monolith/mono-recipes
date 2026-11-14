# WebSocket Chat Demo

A real-time chat application built with the Mono Framework (mono v0.1.1), demonstrating WebSocket communication, the EventBus pubsub pattern for message broadcasting, and module boundaries (typed contracts, a consumer-owned port and adapter, strict mode).

## Features

- **Real-time Messaging**: Bidirectional WebSocket communication
- **Multi-room Support**: Create and join multiple chat rooms
- **Event-driven Architecture**: Messages broadcast via EventBus
- **Message History**: Persistent message history per room
- **User Presence**: Track users joining/leaving rooms

## Why Use WebSockets for Real-time Communication?

WebSockets provide full-duplex, persistent connections—ideal for chat applications:

### 1. Low Latency

Unlike HTTP polling, WebSocket connections stay open:

- **No connection overhead**: Single handshake, then continuous communication
- **Server-initiated messages**: Push updates immediately without client polling
- **Sub-second message delivery**: Perfect for real-time chat

### 2. Efficient Resource Usage

```
HTTP Polling:                    WebSocket:
┌─────────┐  request  ┌─────────┐    ┌─────────┐  connect  ┌─────────┐
│  Client │ --------> │  Server │    │  Client │ <=======> │  Server │
└─────────┘  response └─────────┘    └─────────┘           └─────────┘
     │                     │              │     messages        │
     │   (repeat every     │              │ <------------------ │
     │    1-5 seconds)     │              │ -----------------> │
     ▼                     ▼              │ <------------------ │
```

WebSockets reduce bandwidth and server load by eliminating repeated connection setup.

### 3. Fiber WebSocket Support

The `gofiber/contrib/websocket` package provides:

```go
// WebSocket upgrade middleware
app.Use("/ws", func(c *fiber.Ctx) error {
    if websocket.IsWebSocketUpgrade(c) {
        return c.Next()
    }
    return fiber.ErrUpgradeRequired
})

// WebSocket handler
app.Get("/ws", websocket.New(func(c *websocket.Conn) {
    for {
        _, msg, err := c.ReadMessage()
        if err != nil {
            break
        }
        // Process message...
    }
}))
```

## EventBus Pubsub Pattern for Message Broadcasting

The Mono EventBus enables decoupled message broadcasting:

### How It Works

```
┌──────────────┐  services.chat.*  ┌──────────────┐
│  ws-server   │ ────────────────> │     chat     │
│  (ChatPort → │  typed contracts  │ (RoomStore)  │
│   adapter)   │                   │              │
└──────────────┘                   └──────┬───────┘
       ▲                                  │ EventBus
       │  UserJoined / UserLeft /         │ events.chat.v1.*
       │  ChatMessage                     ▼
       │                           ┌──────────────┐
       └────── broadcast to ────── │ ChatMessage  │──> chat stores the
               room members        │ UserJoined   │    message in history
                                   │ UserLeft     │
                                   └──────────────┘
```

### Event Definitions

Events are declared once, in the emitter's contract package
(`modules/chat/contract/contract.go`), next to the service contracts:

```go
var ChatMessageV1 = helper.EventDefinition[ChatEvent](
    "chat",        // Module name
    "ChatMessage", // Event name
    "v1",          // Version
)

// Publish events (chat module)
contract.ChatMessageV1.Publish(m.eventBus, event, nil)
```

### Consuming Events Through the Contract

Both the chat module (to store history) and the ws-server module (to
broadcast) subscribe with the contract definition, so there is no
`GetEventByName("ChatMessage", "v1", "chat")` string lookup and the payload
arrives decoded:

```go
// chat/module.go: the chat module consumes its own ChatMessage events
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
    return helper.RegisterTypedEventConsumer(registry, contract.ChatMessageV1, m.handleChatMessage, m)
}

func (m *Module) handleChatMessage(_ context.Context, event contract.ChatEvent, _ *mono.Msg) error {
    m.store.AddMessage(event.Message)
    return nil
}

// wsserver/module.go: imports only chat/contract
helper.RegisterTypedEventConsumer(registry, chatcontract.UserJoinedV1, m.broadcast("user_joined"), m)
```

### Why This Pattern?

1. **Loose coupling**: WebSocket handlers don't directly call storage
2. **Extensibility**: Add more consumers (logging, analytics) without changing publisher
3. **Testability**: monotest runs the module on an in-memory EventBus
4. **Persistence**: Events can be durably stored with JetStream

## Module Boundaries

The ws-server module never imports the chat module package. It talks to it
only through `chat/contract`:

- **Contract** (`chat/contract/contract.go`): the DTOs (`Room`, `User`,
  `Message`, the `*ServiceRequest`/`*ServiceResponse` types), nine typed
  request-reply contracts (`CreateRoom`, `GetRoom`, `ListRooms`, `JoinRoom`,
  `LeaveRoom`, `SendMessage`, `GetUser`, `GetHistory`, `GetRoomUsers`, on the
  unchanged subjects `services.chat.<service>`) and the three events.
- **Provider** (`chat/module.go`) registers typed handlers through the contracts:

  ```go
  func (m *Module) RegisterServices(container mono.ServiceContainer) error {
      return errors.Join(
          contract.CreateRoom.Register(container, m.handleCreateRoom),
          contract.JoinRoom.Register(container, m.handleJoinRoom),
          // ...
      )
  }
  ```

- **Consumer** (`wsserver/`): `ports.go` declares `ChatPort`, what the
  handlers need, in the chat contract's DTOs; `adapter.go` implements it over
  the contracts with the container received in `SetDependencyServiceContainer`
  and turns a response's `Error` field into a Go error:

  ```go
  func (a chatAdapter) JoinRoom(ctx context.Context, roomID, userID, username string) (*chatcontract.User, error) {
      resp, err := chatcontract.JoinRoom.Call(ctx, a.container, chatcontract.JoinRoomServiceRequest{
          RoomID: roomID, UserID: userID, Username: username,
      })
      if err != nil {
          return nil, fmt.Errorf("failed to call join-room service: %w", err)
      }
      return resp.User, domainError(resp.Error)
  }
  ```

  `ConsumeServices()` lists all nine contracts, and the framework validates
  them at startup against the chat module's registered services.

`main.go` runs with `mono.WithStrictModuleBoundaries()`, so a boundary
violation (calling an undeclared module's service subject, registering on a
dependency's container) fails instead of only being reported at `Stop`.

### Conformance tests

```bash
go test ./...
```

Each module has a `module_test.go` running `monotest.CheckModule` without NATS:

- `chat`: calls every provided contract through the module's own container,
  checks the message reaches the history through the self-consumed
  `ChatMessage` event, and that all three events are declared and published.
- `wsserver`: a stand-in chat container registers the nine contracts (and
  publishes the chat events like the real module); the exercise drives the
  REST endpoints and a real WebSocket client (join, users, message, history,
  rooms, leave), so every consumed contract and every event consumer runs.

## Scaling Considerations

### Single Instance (Current Demo)

Messages are broadcast to in-memory connections:

```
┌─────────────────────────────────────────┐
│              Single Server              │
│  ┌─────────┐   ┌─────────┐  ┌─────────┐ │
│  │  WS 1   │   │  WS 2   │  │  WS 3   │ │
│  └────┬────┘   └────┬────┘  └────┬────┘ │
│       └────────────┬────────────┘       │
│              sync.Map                   │
└─────────────────────────────────────────┘
```

### Multi-instance (Production)

For horizontal scaling, consider:

1. **Sticky Sessions**: Route WebSocket connections to same server
2. **External Pub/Sub**: Use Redis, NATS, or JetStream for cross-instance messaging
3. **Shared State**: Store room/user state in Redis or database

```
┌─────────────┐         ┌─────────────┐
│  Server 1   │ <====>  │  Server 2   │
│  (WS 1,2)   │  Redis  │  (WS 3,4)   │
└─────────────┘  Pub/Sub└─────────────┘
       │                       │
       └───────────────────────┘
              Load Balancer
```

## API Reference

### WebSocket Protocol

Connect to `ws://localhost:8080/ws` and send JSON messages:

#### Join a Room

```json
{
  "type": "join",
  "payload": {
    "room_id": "lobby",
    "username": "Alice"
  }
}
```

Response:
```json
{
  "type": "joined",
  "payload": {"id": "uuid", "username": "Alice", "room_id": "lobby"}
}
```

#### Send a Message

```json
{
  "type": "message",
  "payload": {
    "content": "Hello, everyone!"
  }
}
```

Broadcast to all room members:
```json
{
  "type": "chat_message",
  "payload": {"room_id": "lobby", "username": "Alice", "content": "Hello, everyone!"}
}
```

#### Leave a Room

```json
{"type": "leave"}
```

#### Get Message History

```json
{"type": "history"}
```

#### Get Room Users

```json
{"type": "users"}
```

#### List Available Rooms

```json
{"type": "rooms"}
```

### REST Endpoints

```bash
# List all rooms
GET /api/v1/rooms

# Create a new room
POST /api/v1/rooms
Content-Type: application/json
{"name": "My Room"}

# Get room message history
GET /api/v1/rooms/:id/history?limit=50

# Health check
GET /health
```

## Project Structure

```
websocket-chat-demo/
├── main.go                        # Application entry point (strict module boundaries)
├── go.mod
├── demo.py                        # Python demo with multiple clients
├── requirements.txt               # Python dependencies
├── README.md
└── modules/
    ├── chat/                      # Chat room service
    │   ├── contract/
    │   │   └── contract.go        # Published port: DTOs, service contracts, events
    │   ├── module.go              # EventBusAware, EventEmitter, EventConsumer, RegisterServices
    │   ├── types.go               # Validation, RoomStore
    │   ├── types_test.go          # Store unit tests
    │   └── module_test.go         # monotest.CheckModule conformance test
    └── wsserver/                  # WebSocket server
        ├── ports.go               # Driven port: ChatPort
        ├── adapter.go             # Adapter over the chat contracts
        ├── module.go              # Fiber server lifecycle, ConsumeServices, event consumers
        ├── handlers.go            # WebSocket and REST handlers
        └── module_test.go         # monotest.CheckModule conformance test
```

## Running the Demo

### Start the Server

```bash
go run .
```

### Run the Python Demo

```bash
# Install dependencies
pip install -r requirements.txt

# Run demo with 3 users sending 5 messages each
python demo.py --users 3 --messages 5
```

The Python demo will:
1. Create/use the lobby room
2. Connect multiple WebSocket clients
3. Have users join the room
4. Simulate a chat conversation
5. Display messages with color-coded usernames

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP/WebSocket server address |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:8080` | CORS allowed origins |

## Key Concepts Demonstrated

1. **WebSocket Connections**: Full-duplex communication with Fiber
2. **EventBusAwareModule**: Receiving EventBus for publishing
3. **EventEmitterModule**: Declaring and publishing chat events
4. **EventConsumerModule**: Consuming events through the contract definitions (history, broadcast)
5. **Typed service contracts**: `chat/contract`, a consumer-owned port and adapter, `ConsumeServices()`
6. **Strict module boundaries**: `mono.WithStrictModuleBoundaries()` and `monotest.CheckModule`
7. **Broadcast Pattern**: Broadcasting to room members via connection map
8. **Graceful Shutdown**: Proper WebSocket connection cleanup
