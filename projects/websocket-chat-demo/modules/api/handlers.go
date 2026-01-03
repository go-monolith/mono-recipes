package api

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/example/websocket-chat-demo/modules/broadcast"
	"github.com/example/websocket-chat-demo/modules/chat"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	maxRoomNameLength = 100
	maxMessageLength  = 4096
	defaultHistoryLimit = 50
)

// setupRoutes configures all HTTP routes.
func (m *APIModule) setupRoutes() {
	// Health check
	m.app.Get("/health", m.healthHandler)

	// WebSocket endpoint
	m.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	m.app.Get("/ws", websocket.New(m.handleWebSocket))

	// REST API v1
	api := m.app.Group("/api/v1")

	// Room management
	api.Get("/rooms", m.listRooms)
	api.Post("/rooms", m.createRoom)
	api.Get("/rooms/:id", m.getRoom)
	api.Get("/rooms/:id/history", m.getHistory)
}

// healthHandler handles GET /health.
func (m *APIModule) healthHandler(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status: "healthy",
		Details: map[string]any{
			"module":            "api",
			"connected_clients": m.hub.ClientCount(),
		},
	})
}

// listRooms handles GET /api/v1/rooms.
func (m *APIModule) listRooms(c *fiber.Ctx) error {
	rooms, err := m.chatAdapter.ListRooms(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "list_failed",
			Message: "Failed to list rooms",
		})
	}

	response := RoomListResponse{
		Rooms: make([]RoomResponse, 0, len(rooms)),
	}
	for _, room := range rooms {
		response.Rooms = append(response.Rooms, RoomResponse{
			ID:        room.ID,
			Name:      room.Name,
			CreatedAt: room.CreatedAt,
			Members:   m.hub.RoomClientCount(room.ID),
		})
	}

	return c.JSON(response)
}

// createRoom handles POST /api/v1/rooms.
func (m *APIModule) createRoom(c *fiber.Ctx) error {
	var req CreateRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Room name is required",
		})
	}

	if len(req.Name) > maxRoomNameLength {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Room name too long (max 100 characters)",
		})
	}

	room, err := m.chatAdapter.CreateRoom(c.UserContext(), req.Name, "api")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "create_failed",
			Message: "Failed to create room",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(RoomResponse{
		ID:        room.ID,
		Name:      room.Name,
		CreatedAt: room.CreatedAt,
	})
}

// getRoom handles GET /api/v1/rooms/:id.
func (m *APIModule) getRoom(c *fiber.Ctx) error {
	roomID := c.Params("id")

	room, err := m.chatAdapter.GetRoom(c.UserContext(), roomID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Room not found",
		})
	}

	return c.JSON(RoomResponse{
		ID:        room.ID,
		Name:      room.Name,
		CreatedAt: room.CreatedAt,
		Members:   m.hub.RoomClientCount(room.ID),
	})
}

// getHistory handles GET /api/v1/rooms/:id/history.
func (m *APIModule) getHistory(c *fiber.Ctx) error {
	roomID := c.Params("id")
	limit := defaultHistoryLimit
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	messages, err := m.chatAdapter.GetHistory(c.UserContext(), roomID, limit)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Room not found",
		})
	}

	response := HistoryResponse{
		RoomID:   roomID,
		Messages: make([]MessageResponse, 0, len(messages)),
	}
	for _, msg := range messages {
		response.Messages = append(response.Messages, MessageResponse{
			ID:        msg.ID,
			RoomID:    msg.RoomID,
			UserID:    msg.UserID,
			Username:  msg.Username,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		})
	}

	return c.JSON(response)
}

// handleWebSocket handles WebSocket connections at /ws.
func (m *APIModule) handleWebSocket(c *websocket.Conn) {
	// Generate client ID and get username from query
	clientID := uuid.New().String()
	username := c.Query("username", "anonymous")

	client := &broadcast.Client{
		ID:       clientID,
		Username: username,
		Conn:     c,
	}

	// Register client with the hub
	m.hub.Register(client)
	defer func() {
		// Leave room if in one
		if client.RoomID != "" {
			_ = m.chatAdapter.LeaveRoom(context.Background(), client.RoomID, clientID, username)
			m.hub.LeaveRoom(clientID)
		}
		m.hub.Unregister(client)
		log.Printf("[api] WebSocket client disconnected: %s (%s)", clientID, username)
	}()

	log.Printf("[api] WebSocket client connected: %s (%s)", clientID, username)

	// Send welcome message
	welcome := chat.WSMessage{
		Type:   "connected",
		UserID: clientID,
	}
	if err := c.WriteJSON(welcome); err != nil {
		log.Printf("[api] Failed to send welcome: %v", err)
		return
	}

	// Message loop
	for {
		_, msgBytes, err := c.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[api] Client %s closed connection", clientID)
			} else {
				log.Printf("[api] Read error from %s: %v", clientID, err)
			}
			break
		}

		var msg chat.WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			m.sendError(c, "Invalid message format")
			continue
		}

		// Handle message based on type
		switch msg.Type {
		case chat.WSTypeJoin:
			m.handleJoin(c, client, msg)
		case chat.WSTypeLeave:
			m.handleLeave(c, client, msg)
		case chat.WSTypeMessage:
			m.handleMessage(c, client, msg)
		case chat.WSTypeHistory:
			m.handleHistoryRequest(c, client, msg)
		case chat.WSTypeMembers:
			m.handleMembersRequest(c, client, msg)
		case chat.WSTypeRoomList:
			m.handleRoomListRequest(c, client)
		default:
			m.sendError(c, "Unknown message type: "+msg.Type)
		}
	}
}

func (m *APIModule) handleJoin(c *websocket.Conn, client *broadcast.Client, msg chat.WSMessage) {
	if msg.RoomID == "" {
		m.sendError(c, "Room ID is required")
		return
	}

	// Leave current room if in one
	if client.RoomID != "" {
		_ = m.chatAdapter.LeaveRoom(context.Background(), client.RoomID, client.ID, client.Username)
		m.hub.LeaveRoom(client.ID)
	}

	// Join new room
	if err := m.chatAdapter.JoinRoom(context.Background(), msg.RoomID, client.ID, client.Username); err != nil {
		m.sendError(c, "Failed to join room: "+err.Error())
		return
	}

	m.hub.JoinRoom(client.ID, msg.RoomID)
	client.RoomID = msg.RoomID

	// Send confirmation
	response := chat.WSMessage{
		Type:   chat.WSTypeJoined,
		RoomID: msg.RoomID,
		UserID: client.ID,
	}
	_ = c.WriteJSON(response)
}

func (m *APIModule) handleLeave(c *websocket.Conn, client *broadcast.Client, _ chat.WSMessage) {
	if client.RoomID == "" {
		m.sendError(c, "Not in a room")
		return
	}

	roomID := client.RoomID
	_ = m.chatAdapter.LeaveRoom(context.Background(), roomID, client.ID, client.Username)
	m.hub.LeaveRoom(client.ID)
	client.RoomID = ""

	// Send confirmation
	response := chat.WSMessage{
		Type:   chat.WSTypeLeft,
		RoomID: roomID,
		UserID: client.ID,
	}
	_ = c.WriteJSON(response)
}

func (m *APIModule) handleMessage(c *websocket.Conn, client *broadcast.Client, msg chat.WSMessage) {
	if client.RoomID == "" {
		m.sendError(c, "Join a room first")
		return
	}

	if msg.Content == "" {
		m.sendError(c, "Message content is required")
		return
	}

	if len(msg.Content) > maxMessageLength {
		m.sendError(c, "Message too long")
		return
	}

	msgID, timestamp, err := m.chatAdapter.SendMessage(
		context.Background(),
		client.RoomID,
		client.ID,
		client.Username,
		msg.Content,
	)
	if err != nil {
		m.sendError(c, "Failed to send message")
		return
	}

	// Send confirmation with message ID and timestamp
	response := chat.WSMessage{
		Type:      chat.WSTypeMessage,
		RoomID:    client.RoomID,
		MessageID: msgID,
		Timestamp: timestamp,
	}
	_ = c.WriteJSON(response)
}

func (m *APIModule) handleHistoryRequest(c *websocket.Conn, client *broadcast.Client, msg chat.WSMessage) {
	roomID := msg.RoomID
	if roomID == "" {
		roomID = client.RoomID
	}
	if roomID == "" {
		m.sendError(c, "Room ID is required")
		return
	}

	limit := 50
	messages, err := m.chatAdapter.GetHistory(context.Background(), roomID, limit)
	if err != nil {
		m.sendError(c, "Failed to get history")
		return
	}

	data, err := json.Marshal(messages)
	if err != nil {
		m.sendError(c, "Failed to encode history response")
		return
	}
	response := chat.WSMessage{
		Type:   chat.WSTypeHistory,
		RoomID: roomID,
		Data:   data,
	}
	_ = c.WriteJSON(response)
}

func (m *APIModule) handleMembersRequest(c *websocket.Conn, client *broadcast.Client, msg chat.WSMessage) {
	roomID := msg.RoomID
	if roomID == "" {
		roomID = client.RoomID
	}
	if roomID == "" {
		m.sendError(c, "Room ID is required")
		return
	}

	members, err := m.chatAdapter.GetRoomMembers(context.Background(), roomID)
	if err != nil {
		m.sendError(c, "Failed to get members")
		return
	}

	data, err := json.Marshal(members)
	if err != nil {
		m.sendError(c, "Failed to encode members response")
		return
	}
	response := chat.WSMessage{
		Type:   chat.WSTypeMembers,
		RoomID: roomID,
		Data:   data,
	}
	_ = c.WriteJSON(response)
}

func (m *APIModule) handleRoomListRequest(c *websocket.Conn, _ *broadcast.Client) {
	rooms, err := m.chatAdapter.ListRooms(context.Background())
	if err != nil {
		m.sendError(c, "Failed to list rooms")
		return
	}

	data, err := json.Marshal(rooms)
	if err != nil {
		m.sendError(c, "Failed to encode room list response")
		return
	}
	response := chat.WSMessage{
		Type: chat.WSTypeRoomList,
		Data: data,
	}
	_ = c.WriteJSON(response)
}

func (m *APIModule) sendError(c *websocket.Conn, message string) {
	response := chat.WSMessage{
		Type:  chat.WSTypeError,
		Error: message,
	}
	_ = c.WriteJSON(response)
}
