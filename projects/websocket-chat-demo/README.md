# WebSocket Chat Demo

A real-time chat application built with the Mono Framework, demonstrating WebSocket communication and the EventBus pubsub pattern for message broadcasting.

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
┌──────────────┐     EventBus      ┌──────────────┐
│  Chat Module │ ────────────────> │  Chat Module │
│  (Publisher) │   ChatMessage     │  (Consumer)  │
│              │   UserJoined      │              │
│              │   UserLeft        │              │
└──────────────┘                   └──────────────┘
                                          │
                                          ▼
                                   Store in history

┌──────────────┐                   ┌──────────────┐
│  WebSocket   │ ←── Read from ─── │  Room Store  │
│  Handler     │     store and     │  (History)   │
│              │     broadcast     │              │
└──────────────┘                   └──────────────┘
```

### Event Definitions

```go
// Define typed events
var ChatMessageV1 = helper.EventDefinition[ChatEvent](
    "chat",        // Module name
    "ChatMessage", // Event name
    "v1",          // Version
)

// Publish events
ChatMessageV1.Publish(eventBus, event, nil)
```

### Self-consuming Events

The chat module both emits and consumes events:

```go
// Emit events when messages are sent
func (m *Module) SendMessage(userID, content string) error {
    ChatMessageV1.Publish(m.eventBus, event, nil)
}

// Consume events to store in history
func (m *Module) RegisterEventConsumers(registry mono.EventRegistry) error {
    registry.RegisterEventConsumer(msgDef, m.handleChatMessage, m)
}
```

### Why This Pattern?

1. **Loose coupling**: WebSocket handlers don't directly call storage
2. **Extensibility**: Add more consumers (logging, analytics) without changing publisher
3. **Testability**: Mock EventBus for unit testing
4. **Persistence**: Events can be durably stored with JetStream

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
├── main.go                        # Application entry point
├── go.mod
├── demo.py                        # Python demo with multiple clients
├── requirements.txt               # Python dependencies
├── README.md
└── modules/
    ├── chat/                      # Chat room service
    │   ├── module.go              # EventBusAware, EventEmitter, EventConsumer
    │   ├── types.go               # Room, User, Message, RoomStore
    │   ├── events.go              # Event definitions
    │   └── types_test.go          # Unit tests
    └── wsserver/                  # WebSocket server
        ├── module.go              # Fiber server lifecycle
        └── handlers.go            # WebSocket and REST handlers
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
4. **EventConsumerModule**: Consuming events for message history
5. **Broadcast Pattern**: Broadcasting to room members via connection map
6. **Graceful Shutdown**: Proper WebSocket connection cleanup
