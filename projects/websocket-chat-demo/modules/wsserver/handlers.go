package wsserver

import (
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
	refillRate int       // tokens per second
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

// Handlers contains HTTP and WebSocket handlers.
type Handlers struct {
	chatModule   *chat.Module
	connections  sync.Map // userID -> *websocket.Conn
	rateLimiters sync.Map // userID -> *rateLimiter
	logger       *slog.Logger
}

// NewHandlers creates a new handlers instance.
func NewHandlers(chatModule *chat.Module) *Handlers {
	return &Handlers{
		chatModule: chatModule,
		logger:     slog.Default(),
	}
}

// HandleWebSocket handles WebSocket connections.
func (h *Handlers) HandleWebSocket(c *websocket.Conn) {
	userID := uuid.New().String()
	h.connections.Store(userID, c)

	// Create rate limiter for this user
	h.rateLimiters.Store(userID, newRateLimiter(burstSize, messagesPerSecond))

	defer func() {
		h.connections.Delete(userID)
		h.rateLimiters.Delete(userID)
		h.chatModule.LeaveRoom(userID)
		c.Close()
	}()

	h.logger.Info("WebSocket connected", "userID", userID)

	for {
		_, msgBytes, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error", "userID", userID, "error", err)
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			h.sendError(c, "Invalid message format")
			continue
		}

		h.handleMessage(c, userID, msg)
	}

	h.logger.Info("WebSocket disconnected", "userID", userID)
}

// handleMessage processes incoming WebSocket messages.
func (h *Handlers) handleMessage(c *websocket.Conn, userID string, msg WebSocketMessage) {
	switch msg.Type {
	case "join":
		h.handleJoin(c, userID, msg.Payload)
	case "leave":
		h.handleLeave(c, userID)
	case "message":
		h.handleChatMessage(c, userID, msg.Payload)
	case "history":
		h.handleHistory(c, userID)
	case "users":
		h.handleUsers(c, userID)
	case "rooms":
		h.handleRooms(c)
	default:
		h.sendError(c, "Unknown message type: "+msg.Type)
	}
}

// handleJoin processes join room requests.
func (h *Handlers) handleJoin(c *websocket.Conn, userID string, payload json.RawMessage) {
	var req JoinPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "Invalid join payload")
		return
	}

	if req.RoomID == "" || req.Username == "" {
		h.sendError(c, "room_id and username are required")
		return
	}

	user, err := h.chatModule.JoinRoom(req.RoomID, userID, req.Username)
	if err != nil {
		h.sendError(c, err.Error())
		return
	}

	// Send confirmation
	h.sendMessage(c, "joined", user)

	// Broadcast to room that user joined
	h.broadcastToRoom(req.RoomID, "user_joined", chat.Message{
		RoomID:   req.RoomID,
		UserID:   userID,
		Username: req.Username,
		Content:  req.Username + " joined the room",
		Type:     "join",
	})
}

// handleLeave processes leave room requests.
func (h *Handlers) handleLeave(c *websocket.Conn, userID string) {
	user, exists := h.chatModule.Store().GetUser(userID)
	if !exists {
		h.sendError(c, "Not in a room")
		return
	}

	roomID := user.RoomID
	username := user.Username
	h.chatModule.LeaveRoom(userID)

	// Send confirmation
	h.sendMessage(c, "left", map[string]string{"room_id": roomID})

	// Broadcast to room that user left
	h.broadcastToRoom(roomID, "user_left", chat.Message{
		RoomID:   roomID,
		UserID:   userID,
		Username: username,
		Content:  username + " left the room",
		Type:     "leave",
	})
}

// handleChatMessage processes chat messages.
func (h *Handlers) handleChatMessage(c *websocket.Conn, userID string, payload json.RawMessage) {
	// Rate limit check
	if limiterVal, ok := h.rateLimiters.Load(userID); ok {
		limiter := limiterVal.(*rateLimiter)
		if !limiter.allow() {
			h.sendError(c, "Rate limit exceeded, please slow down")
			return
		}
	}

	var req MessagePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		h.sendError(c, "Invalid message payload")
		return
	}

	if req.Content == "" {
		h.sendError(c, "Message content is required")
		return
	}

	user, exists := h.chatModule.Store().GetUser(userID)
	if !exists {
		h.sendError(c, "Not in a room")
		return
	}

	if err := h.chatModule.SendMessage(userID, req.Content); err != nil {
		h.sendError(c, err.Error())
		return
	}

	// Broadcast message to room
	h.broadcastToRoom(user.RoomID, "chat_message", chat.Message{
		RoomID:   user.RoomID,
		UserID:   userID,
		Username: user.Username,
		Content:  req.Content,
		Type:     "message",
	})
}

// handleHistory sends message history to the client.
func (h *Handlers) handleHistory(c *websocket.Conn, userID string) {
	user, exists := h.chatModule.Store().GetUser(userID)
	if !exists {
		h.sendError(c, "Not in a room")
		return
	}

	history := h.chatModule.GetHistory(user.RoomID, 50)
	h.sendMessage(c, "history", history)
}

// handleUsers sends list of users in the room.
func (h *Handlers) handleUsers(c *websocket.Conn, userID string) {
	user, exists := h.chatModule.Store().GetUser(userID)
	if !exists {
		h.sendError(c, "Not in a room")
		return
	}

	users := h.chatModule.GetRoomUsers(user.RoomID)
	h.sendMessage(c, "users", users)
}

// handleRooms sends list of available rooms.
func (h *Handlers) handleRooms(c *websocket.Conn) {
	rooms := h.chatModule.ListRooms()
	h.sendMessage(c, "rooms", rooms)
}

// broadcastToRoom sends a message to all users in a room.
func (h *Handlers) broadcastToRoom(roomID string, msgType string, data any) {
	users := h.chatModule.GetRoomUsers(roomID)

	for _, user := range users {
		if conn, ok := h.connections.Load(user.ID); ok {
			if ws, ok := conn.(*websocket.Conn); ok {
				h.sendMessage(ws, msgType, data)
			}
		}
	}
}

// sendMessage sends a typed message to a WebSocket connection.
func (h *Handlers) sendMessage(c *websocket.Conn, msgType string, data any) {
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

	if err := c.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		h.logger.Error("Failed to send WebSocket message", "error", err)
	}
}

// sendError sends an error message to a WebSocket connection.
func (h *Handlers) sendError(c *websocket.Conn, errMsg string) {
	msg := WebSocketMessage{
		Type:  "error",
		Error: errMsg,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal error message", "error", err)
		return
	}

	if err := c.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		h.logger.Error("Failed to send error message", "error", err)
	}
}

// REST Handlers

// ListRooms handles room listing requests (GET /api/v1/rooms).
func (h *Handlers) ListRooms(c *fiber.Ctx) error {
	rooms := h.chatModule.ListRooms()
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

	room, err := h.chatModule.CreateRoom(req.Name)
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

	if _, exists := h.chatModule.GetRoom(roomID); !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Room not found",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}

	history := h.chatModule.GetHistory(roomID, limit)
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
