package wsserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/example/websocket-chat-demo/modules/chat"
)

// Rate limiting constants
const (
	messagesPerSecond = 10
	burstSize         = 20
)

// WebSocketMessage represents a message sent over WebSocket.
type WebSocketMessage struct {
	Type    string          `json:"type"` // "join", "leave", "message", "history", "users", "rooms"
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// JoinPayload is the payload for joining a room.
type JoinPayload struct {
	RoomID   string `json:"room_id"`
	Username string `json:"username"`
}

// MessagePayload is the payload for sending a message.
type MessagePayload struct {
	Content string `json:"content"`
}

// rateLimiter implements a simple token bucket rate limiter.
type rateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate int // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func newRateLimiter(maxTokens, refillRate int) *rateLimiter {
	return &rateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *rateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * r.refillRate
	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// connWrapper wraps a WebSocket connection with a mutex for thread-safe writes.
type connWrapper struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// WriteMessage writes a message to the WebSocket connection in a thread-safe manner.
func (cw *connWrapper) WriteMessage(messageType int, data []byte) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return cw.conn.WriteMessage(messageType, data)
}

// Handlers contains HTTP and WebSocket handlers.
type Handlers struct {
	chatAdapter  *chat.ServiceAdapter
	connections  sync.Map // userID -> *connWrapper
	userRooms    sync.Map // userID -> roomID (for broadcast lookups)
	rateLimiters sync.Map // userID -> *rateLimiter
	logger       *slog.Logger
}

// NewHandlers creates a new handlers instance.
func NewHandlers(chatAdapter *chat.ServiceAdapter) *Handlers {
	return &Handlers{
		chatAdapter: chatAdapter,
		logger:      slog.Default(),
	}
}

// HandleWebSocket handles WebSocket connections.
func (h *Handlers) HandleWebSocket(c *websocket.Conn) {
	userID := uuid.New().String()
	cw := &connWrapper{conn: c}
	h.connections.Store(userID, cw)

	// Create rate limiter for this user
	h.rateLimiters.Store(userID, newRateLimiter(burstSize, messagesPerSecond))

	defer func() {
		// Only leave room if user actually joined one
		if _, inRoom := h.userRooms.Load(userID); inRoom {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := h.chatAdapter.LeaveRoom(ctx, userID); err != nil {
				h.logger.Error("Failed to leave room on disconnect", "userID", userID, "error", err)
			}
		}
		h.connections.Delete(userID)
		h.userRooms.Delete(userID)
		h.rateLimiters.Delete(userID)
		c.Close()
	}()

	h.logger.Info("WebSocket connected", "userID", userID)

	for {
		_, msgBytes, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error", "userID", userID, "error", err)
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			h.sendErrorToWrapper(cw, "Invalid message format")
			continue
		}

		h.handleMessage(cw, userID, msg)
	}

	h.logger.Info("WebSocket disconnected", "userID", userID)
}

// handleMessage processes incoming WebSocket messages.
func (h *Handlers) handleMessage(cw *connWrapper, userID string, msg WebSocketMessage) {
	switch msg.Type {
	case "join":
		h.handleJoin(cw, userID, msg.Payload)
	case "leave":
		h.handleLeave(cw, userID)
	case "message":
		h.handleChatMessage(cw, userID, msg.Payload)
	case "history":
		h.handleHistory(cw, userID)
	case "users":
		h.handleUsers(cw, userID)
	case "rooms":
		h.handleRooms(cw)
	default:
		h.sendErrorToWrapper(cw, "Unknown message type: "+msg.Type)
	}
}

// handleJoin processes join room requests.
func (h *Handlers) handleJoin(cw *connWrapper, userID string, payload json.RawMessage) {
	var req JoinPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendErrorToWrapper(cw, "Invalid join payload")
		return
	}

	if req.RoomID == "" || req.Username == "" {
		h.sendErrorToWrapper(cw, "room_id and username are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.chatAdapter.JoinRoom(ctx, req.RoomID, userID, req.Username)
	if err != nil {
		h.sendErrorToWrapper(cw, err.Error())
		return
	}

	// Track user's room for broadcasts
	h.userRooms.Store(userID, req.RoomID)

	// Send confirmation to this user
	h.sendMessageToWrapper(cw, "joined", user)

	// Note: Broadcast is handled by event consumer in module.go
}

// handleLeave processes leave room requests.
func (h *Handlers) handleLeave(cw *connWrapper, userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, exists, err := h.chatAdapter.GetUser(ctx, userID)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get user")
		return
	}
	if !exists {
		h.sendErrorToWrapper(cw, "Not in a room")
		return
	}

	roomID := user.RoomID
	if err := h.chatAdapter.LeaveRoom(ctx, userID); err != nil {
		h.sendErrorToWrapper(cw, err.Error())
		return
	}

	// Clear user's room tracking
	h.userRooms.Delete(userID)

	// Send confirmation
	h.sendMessageToWrapper(cw, "left", map[string]string{"room_id": roomID})

	// Note: Broadcast is handled by event consumer in module.go
}

// handleChatMessage processes chat messages.
func (h *Handlers) handleChatMessage(cw *connWrapper, userID string, payload json.RawMessage) {
	// Rate limit check
	if limiterVal, ok := h.rateLimiters.Load(userID); ok {
		limiter := limiterVal.(*rateLimiter)
		if !limiter.allow() {
			h.sendErrorToWrapper(cw, "Rate limit exceeded, please slow down")
			return
		}
	}

	var req MessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendErrorToWrapper(cw, "Invalid message payload")
		return
	}

	if req.Content == "" {
		h.sendErrorToWrapper(cw, "Message content is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, exists, err := h.chatAdapter.GetUser(ctx, userID)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get user")
		return
	}
	if !exists {
		h.sendErrorToWrapper(cw, "Not in a room")
		return
	}

	if err := h.chatAdapter.SendMessage(ctx, userID, req.Content); err != nil {
		h.sendErrorToWrapper(cw, err.Error())
		return
	}

	// Broadcast is handled by event consumer in module.go
}

// handleHistory sends message history to the client.
func (h *Handlers) handleHistory(cw *connWrapper, userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, exists, err := h.chatAdapter.GetUser(ctx, userID)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get user")
		return
	}
	if !exists {
		h.sendErrorToWrapper(cw, "Not in a room")
		return
	}

	history, err := h.chatAdapter.GetHistory(ctx, user.RoomID, 50)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get history")
		return
	}
	h.sendMessageToWrapper(cw, "history", history)
}

// handleUsers sends list of users in the room.
func (h *Handlers) handleUsers(cw *connWrapper, userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, exists, err := h.chatAdapter.GetUser(ctx, userID)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get user")
		return
	}
	if !exists {
		h.sendErrorToWrapper(cw, "Not in a room")
		return
	}

	users, err := h.chatAdapter.GetRoomUsers(ctx, user.RoomID)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get users")
		return
	}
	h.sendMessageToWrapper(cw, "users", users)
}

// handleRooms sends list of available rooms.
func (h *Handlers) handleRooms(cw *connWrapper) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rooms, err := h.chatAdapter.ListRooms(ctx)
	if err != nil {
		h.sendErrorToWrapper(cw, "Failed to get rooms")
		return
	}
	h.sendMessageToWrapper(cw, "rooms", rooms)
}

// BroadcastToRoom sends a message to all users in a room.
// This is exported for use by the module's event consumers.
func (h *Handlers) BroadcastToRoom(roomID string, msgType string, data any) {
	// Iterate over all connections and send to those in the room
	h.connections.Range(func(key, value any) bool {
		userID := key.(string)
		cw := value.(*connWrapper)

		// Check if this user is in the target room
		if room, ok := h.userRooms.Load(userID); ok {
			if room.(string) == roomID {
				h.sendMessageToWrapper(cw, msgType, data)
			}
		}
		return true
	})
}

// sendMessageToWrapper sends a typed message to a WebSocket connection using the thread-safe wrapper.
func (h *Handlers) sendMessageToWrapper(cw *connWrapper, msgType string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("Failed to marshal message", "error", err)
		return
	}

	msg := WebSocketMessage{
		Type:    msgType,
		Payload: payload,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal WebSocket message", "error", err)
		return
	}

	if err := cw.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		h.logger.Error("Failed to send WebSocket message", "error", err)
	}
}

// sendErrorToWrapper sends an error message to a WebSocket connection using the thread-safe wrapper.
func (h *Handlers) sendErrorToWrapper(cw *connWrapper, errMsg string) {
	msg := WebSocketMessage{
		Type:  "error",
		Error: errMsg,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal error message", "error", err)
		return
	}

	if err := cw.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		h.logger.Error("Failed to send error message", "error", err)
	}
}

// REST Handlers

// ListRooms handles room listing requests (GET /api/v1/rooms).
func (h *Handlers) ListRooms(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	rooms, err := h.chatAdapter.ListRooms(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list rooms",
		})
	}
	return c.JSON(fiber.Map{
		"rooms": rooms,
		"total": len(rooms),
	})
}

// CreateRoom handles room creation requests (POST /api/v1/rooms).
func (h *Handlers) CreateRoom(c *fiber.Ctx) error {
	var req chat.CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	room, err := h.chatAdapter.CreateRoom(ctx, req.Name)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(room)
}

// GetRoomHistory handles room history requests (GET /api/v1/rooms/:id/history).
func (h *Handlers) GetRoomHistory(c *fiber.Ctx) error {
	roomID := c.Params("id")
	if roomID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Room ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	_, exists, err := h.chatAdapter.GetRoom(ctx, roomID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get room",
		})
	}
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}

	history, err := h.chatAdapter.GetHistory(ctx, roomID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get history",
		})
	}
	return c.JSON(fiber.Map{
		"room_id":  roomID,
		"messages": history,
		"total":    len(history),
	})
}

// HealthCheck handles health check requests (GET /health).
func (h *Handlers) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "websocket-chat-demo",
	})
}
