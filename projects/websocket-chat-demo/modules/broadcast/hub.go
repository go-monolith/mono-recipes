package broadcast

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

// Client represents a connected WebSocket client.
type Client struct {
	ID       string
	Username string
	RoomID   string
	Conn     *websocket.Conn
}

// Hub manages WebSocket connections and message broadcasting.
type Hub struct {
	clients    map[string]*Client          // clientID -> Client
	rooms      map[string]map[string]bool  // roomID -> set of clientIDs
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	done       chan struct{}
	mu         sync.RWMutex
}

// BroadcastMessage represents a message to broadcast.
type BroadcastMessage struct {
	RoomID  string
	Type    string
	Payload any
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop. It accepts a context for graceful shutdown.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("[hub] Shutting down...")
			h.closeAllClients()
			close(h.done)
			return
		case client := <-h.register:
			h.handleRegister(client)
		case client := <-h.unregister:
			h.handleUnregister(client)
		case msg := <-h.broadcast:
			h.handleBroadcast(msg)
		}
	}
}

// Wait blocks until the hub has stopped.
func (h *Hub) Wait() {
	<-h.done
}

// closeAllClients closes all connected client connections.
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		_ = client.Conn.Close()
	}
	h.clients = make(map[string]*Client)
	h.rooms = make(map[string]map[string]bool)
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	if client.RoomID != "" {
		if h.rooms[client.RoomID] == nil {
			h.rooms[client.RoomID] = make(map[string]bool)
		}
		h.rooms[client.RoomID][client.ID] = true
	}
	log.Printf("[hub] Client %s (%s) registered", client.ID, client.Username)
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		if client.RoomID != "" && h.rooms[client.RoomID] != nil {
			delete(h.rooms[client.RoomID], client.ID)
			if len(h.rooms[client.RoomID]) == 0 {
				delete(h.rooms, client.RoomID)
			}
		}
		log.Printf("[hub] Client %s (%s) unregistered", client.ID, client.Username)
	}
}

func (h *Hub) handleBroadcast(msg *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("[hub] Failed to marshal broadcast message: %v", err)
		return
	}

	if msg.RoomID == "" {
		// Broadcast to all clients
		for _, client := range h.clients {
			h.sendToClient(client, data)
		}
	} else {
		// Broadcast to room members only
		if clientIDs, ok := h.rooms[msg.RoomID]; ok {
			for clientID := range clientIDs {
				if client, ok := h.clients[clientID]; ok {
					h.sendToClient(client, data)
				}
			}
		}
	}
}

func (h *Hub) sendToClient(client *Client, data []byte) {
	if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[hub] Failed to send to client %s: %v", client.ID, err)
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all clients in a room.
func (h *Hub) Broadcast(roomID, msgType string, payload any) {
	h.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Type:    msgType,
		Payload: payload,
	}
}

// JoinRoom moves a client to a specific room.
func (h *Hub) JoinRoom(clientID, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[clientID]
	if !ok {
		return
	}

	// Leave old room if any
	if client.RoomID != "" && h.rooms[client.RoomID] != nil {
		delete(h.rooms[client.RoomID], clientID)
		if len(h.rooms[client.RoomID]) == 0 {
			delete(h.rooms, client.RoomID)
		}
	}

	// Join new room
	client.RoomID = roomID
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[string]bool)
	}
	h.rooms[roomID][clientID] = true
	log.Printf("[hub] Client %s joined room %s", clientID, roomID)
}

// LeaveRoom removes a client from their current room.
func (h *Hub) LeaveRoom(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[clientID]
	if !ok || client.RoomID == "" {
		return
	}

	if h.rooms[client.RoomID] != nil {
		delete(h.rooms[client.RoomID], clientID)
		if len(h.rooms[client.RoomID]) == 0 {
			delete(h.rooms, client.RoomID)
		}
	}
	log.Printf("[hub] Client %s left room %s", clientID, client.RoomID)
	client.RoomID = ""
}

// GetClient returns a client by ID.
func (h *Hub) GetClient(clientID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[clientID]
}

// GetRoomClients returns all clients in a room.
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var clients []*Client
	if clientIDs, ok := h.rooms[roomID]; ok {
		for clientID := range clientIDs {
			if client, ok := h.clients[clientID]; ok {
				clients = append(clients, client)
			}
		}
	}
	return clients
}

// ClientCount returns the total number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// RoomClientCount returns the number of clients in a room.
func (h *Hub) RoomClientCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[roomID]; ok {
		return len(clients)
	}
	return 0
}
