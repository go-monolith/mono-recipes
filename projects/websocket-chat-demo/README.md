# WebSocket Chat Demo

A real-time chat application built with the Mono framework, demonstrating **Fiber WebSocket support** and **EventBus pubsub** for message broadcasting.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     WebSocket Clients                            │
│    ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐          │
│    │ User 1  │  │ User 2  │  │ User 3  │  │ User N  │          │
│    └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘          │
└─────────┼────────────┼────────────┼────────────┼────────────────┘
          │            │            │            │
          └────────────┴────────────┴────────────┘
                              │ WebSocket (ws://localhost:3000/ws)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API Module (Fiber)                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  WebSocket Handler: Connection management, message routing   ││
│  │  REST Endpoints: Room CRUD, message history                  ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────────┘
                              │ DependentModule
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                Chat Module (ServiceProviderModule)               │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Services: CreateRoom, JoinRoom, SendMessage, GetHistory     ││
│  │  EventEmitter: MessageSent, UserJoined, UserLeft, RoomCreated││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────────┘
                              │ Events via NATS JetStream
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│            Broadcast Module (EventConsumerModule)                │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  WebSocket Hub: Client registry, room-based broadcasting     ││
│  │  Event Handlers: Forward events to connected clients         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Why Use WebSockets for Real-Time Communication

### The Problem with HTTP Polling

Traditional HTTP-based chat implementations use polling:
```
Client                    Server
  │ ──── GET /messages ──────> │  (every 1-2 seconds)
  │ <──── [messages] ────────  │
  │ ──── GET /messages ──────> │
  │ <──── [no new messages] ── │
  │ ──── GET /messages ──────> │
  │ <──── [1 new message] ──── │
```

**Problems:**
- Wasted bandwidth (most requests return empty)
- Latency (up to polling interval delay)
- Server load (N clients × M requests/second)
- Not truly real-time

### The WebSocket Solution

WebSockets provide a persistent, bidirectional connection:
```
Client                    Server
  │ ──── Upgrade to WS ──────> │  (once)
  │ <──── Connection OK ─────  │
  │                            │
  │ <──── New message ──────── │  (instant push)
  │ ──── Send message ───────> │  (instant send)
  │ <──── Broadcast ─────────  │  (instant delivery)
```

**Benefits:**
- **True real-time**: Sub-millisecond message delivery
- **Efficient**: Single connection, no polling overhead
- **Bidirectional**: Server can push to clients anytime
- **Low latency**: No HTTP request/response cycle

### WebSocket in Fiber

This demo uses [gofiber/contrib/websocket](https://github.com/gofiber/contrib/tree/main/websocket):

```go
// Upgrade HTTP to WebSocket
m.app.Get("/ws", websocket.New(m.handleWebSocket))

// Handle WebSocket messages
func (m *APIModule) handleWebSocket(c *websocket.Conn) {
    for {
        _, msg, err := c.ReadMessage()
        // Process message...
        c.WriteJSON(response)
    }
}
```

## EventBus Pubsub Pattern for Message Broadcasting

### How It Works

The EventBus pattern decouples message producers from consumers:

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Chat Module    │     │   NATS JetStream │     │ Broadcast Module │
│ (EventEmitter)   │────>│    (EventBus)    │────>│ (EventConsumer)  │
└──────────────────┘     └──────────────────┘     └──────────────────┘
        │                                                  │
   SendMessage()                                    handleMessageSent()
        │                                                  │
        ▼                                                  ▼
  MessageSentEvent ──────────────────────────────> Broadcast to WS Hub
```

### Event Flow Example

1. **User sends message** via WebSocket
2. **API Module** calls `chatAdapter.SendMessage()`
3. **Chat Module** stores message and emits `MessageSentEvent`
4. **EventBus** (NATS JetStream) delivers event to subscribers
5. **Broadcast Module** receives event and broadcasts to WebSocket hub
6. **Hub** sends message to all clients in the room

### Benefits of EventBus Pattern

1. **Decoupled Architecture**
   - Chat logic doesn't know about WebSocket broadcasting
   - Easy to add new consumers (logging, analytics, notifications)
   - Modules can be developed and tested independently

2. **Scalability**
   - Events can be processed by multiple consumers in parallel
   - Horizontal scaling with message queuing
   - Backpressure handling built into NATS

3. **Reliability**
   - Events are persisted in JetStream
   - At-least-once delivery guarantees
   - Replay capability for recovery

4. **Flexibility**
   - Add new event types without changing existing code
   - Different delivery semantics per consumer
   - Easy to integrate external services

### Event Definitions

```go
// Domain events for chat
var (
    MessageSentV1 = helper.EventDefinition[MessageSentEvent](
        "chat", "MessageSent", "v1",
    )
    UserJoinedV1 = helper.EventDefinition[UserJoinedEvent](
        "chat", "UserJoined", "v1",
    )
    UserLeftV1 = helper.EventDefinition[UserLeftEvent](
        "chat", "UserLeft", "v1",
    )
)
```

## Scaling Considerations

### Single Instance (This Demo)

Current architecture works well for:
- Development and testing
- Small deployments (< 1000 concurrent users)
- Simple use cases

```
┌─────────────────────────────────────┐
│         Single Application          │
│  ┌─────┐  ┌─────┐  ┌───────────┐   │
│  │ API │  │Chat │  │ Broadcast │   │
│  └──┬──┘  └──┬──┘  └─────┬─────┘   │
│     │        │           │         │
│     └────────┴───────────┘         │
│              │                      │
│         In-Memory Hub               │
└─────────────────────────────────────┘
```

### Multi-Instance Scaling

For production deployments with multiple instances:

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Instance 1    │  │   Instance 2    │  │   Instance 3    │
│   (WS Hub A)    │  │   (WS Hub B)    │  │   (WS Hub C)    │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         │                    │                    │
         └────────────────────┴────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   Load Balancer   │
                    │ (Sticky Sessions) │
                    └─────────┬─────────┘
                              │
                    ┌─────────▼─────────┐
                    │  NATS JetStream   │
                    │ (Shared EventBus) │
                    └───────────────────┘
```

**Key Considerations:**

1. **Sticky Sessions Required**
   - WebSocket connections must stay on the same instance
   - Use cookie-based or IP-based session affinity
   - Load balancer examples: nginx, HAProxy, AWS ALB

2. **Cross-Instance Broadcasting**
   - Each instance has its own WebSocket hub
   - NATS delivers events to ALL instances
   - Each hub broadcasts to its local clients

3. **State Synchronization**
   - Room membership tracked in shared storage (Redis/NATS KV)
   - Message history in persistent storage
   - User presence via heartbeats

### Production Recommendations

| Concern | Solution |
|---------|----------|
| Sticky Sessions | nginx `ip_hash` or cookie affinity |
| Cross-Instance Pubsub | NATS JetStream (already integrated) |
| Room State | NATS KV or Redis |
| Message Persistence | PostgreSQL or MongoDB |
| Connection Limits | Configure Fiber's `Concurrency` setting |
| Heartbeats | WebSocket ping/pong (built-in) |

## Mono Framework Patterns Demonstrated

### 1. ServiceProviderModule
Chat module exposes services for room and message management:
```go
helper.RegisterTypedRequestReplyService(
    registry, ServiceSendMessage, m.handleSendMessage,
)
```

### 2. EventEmitterModule
Chat module publishes events when actions occur:
```go
func (m *ChatModule) RegisterEventEmitters(registry mono.EventRegistry) error {
    return registry.RegisterEmitter(
        events.MessageSentV1,
        events.UserJoinedV1,
        events.UserLeftV1,
    )
}
```

### 3. EventConsumerModule
Broadcast module subscribes to events for WebSocket broadcasting:
```go
func (m *BroadcastModule) RegisterEventConsumers(registry mono.EventRegistry) error {
    return helper.RegisterTypedEventConsumer(
        registry, events.MessageSentV1, m.handleMessageSent, m,
    )
}
```

### 4. DependentModule
API module depends on chat and broadcast modules:
```go
func (m *APIModule) Dependencies() []string {
    return []string{"chat", "broadcast"}
}
```

### 5. HealthCheckableModule
All modules implement health checks:
```go
func (m *BroadcastModule) Health(_ context.Context) mono.HealthStatus {
    return mono.HealthStatus{
        Healthy: true,
        Details: map[string]any{
            "connected_clients": m.hub.ClientCount(),
        },
    }
}
```

## API Reference

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check with connection stats |
| GET | `/api/v1/rooms` | List all chat rooms |
| POST | `/api/v1/rooms` | Create a new room |
| GET | `/api/v1/rooms/:id` | Get room details |
| GET | `/api/v1/rooms/:id/history` | Get message history |

### WebSocket Protocol

Connect to: `ws://localhost:3000/ws?username=yourname`

**Client → Server Messages:**

```json
// Join a room
{"type": "join", "room_id": "room-uuid"}

// Leave current room
{"type": "leave"}

// Send a message (must be in a room)
{"type": "message", "content": "Hello!"}

// Get message history
{"type": "history", "room_id": "room-uuid"}

// Get room members
{"type": "members", "room_id": "room-uuid"}

// List all rooms
{"type": "room_list"}
```

**Server → Client Messages:**

```json
// Welcome (on connect)
{"type": "connected", "user_id": "your-uuid"}

// Joined room confirmation
{"type": "joined", "room_id": "room-uuid"}

// Left room confirmation
{"type": "left", "room_id": "room-uuid"}

// Message broadcast
{
  "type": "message",
  "room_id": "room-uuid",
  "message_id": "msg-uuid",
  "user_id": "sender-uuid",
  "username": "sender",
  "content": "Hello!",
  "timestamp": "2024-01-15T10:30:00Z"
}

// User joined notification
{"type": "user_joined", "room_id": "...", "username": "..."}

// User left notification
{"type": "user_left", "room_id": "...", "username": "..."}

// Error
{"type": "error", "error": "Error message"}
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Python 3.8+ (for demo script)

### Running the Application

1. Start NATS JetStream:
   ```bash
   docker-compose up -d
   ```

2. Run the application:
   ```bash
   go run main.go
   ```

3. Install Python dependencies:
   ```bash
   pip install -r requirements.txt
   ```

4. Run the demo:
   ```bash
   python demo.py
   ```

### Demo Script Options

```bash
# Default: 3 users, 5 messages each
python demo.py

# Custom number of users
python demo.py --users 5

# Custom message count
python demo.py --messages 10

# Custom room name
python demo.py --room "developers"

# Connect to remote server
python demo.py --host 192.168.1.100 --port 3000

# Adjust message delay
python demo.py --delay 0.5
```

## Project Structure

```
websocket-chat-demo/
├── domain/
│   └── chat/
│       └── entity.go          # Domain entities (Room, Message, User)
├── events/
│   └── chat_events.go         # Event definitions
├── modules/
│   ├── chat/
│   │   ├── service.go         # Business logic
│   │   ├── module.go          # ServiceProviderModule + EventEmitter
│   │   ├── adapter.go         # Cross-module adapter
│   │   └── types.go           # Request/response types
│   ├── broadcast/
│   │   ├── hub.go             # WebSocket connection hub
│   │   └── module.go          # EventConsumerModule
│   └── api/
│       ├── module.go          # Fiber HTTP/WS module
│       ├── handlers.go        # HTTP and WebSocket handlers
│       └── types.go           # API types
├── main.go                    # Entry point
├── docker-compose.yml         # NATS JetStream
├── demo.py                    # Python WebSocket demo
├── requirements.txt           # Python dependencies
└── README.md
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `PORT` | `3000` | HTTP/WebSocket server port |

## License

MIT License - See LICENSE file for details.
