package chat

import (
	"errors"
	"sync"
	"time"
	"unicode/utf8"
)

// Validation constants
const (
	MaxUsernameLength = 50
	MaxRoomNameLength = 100
	MaxMessageLength  = 5000
)

// Validation errors
var (
	ErrUsernameEmpty     = errors.New("username cannot be empty")
	ErrUsernameTooLong   = errors.New("username exceeds maximum length")
	ErrUsernameInvalid   = errors.New("username contains invalid characters")
	ErrRoomNameEmpty     = errors.New("room name cannot be empty")
	ErrRoomNameTooLong   = errors.New("room name exceeds maximum length")
	ErrRoomNameInvalid   = errors.New("room name contains invalid characters")
	ErrMessageEmpty      = errors.New("message content cannot be empty")
	ErrMessageTooLong    = errors.New("message exceeds maximum length")
	ErrMessageInvalid    = errors.New("message contains invalid characters")
	ErrRoomAlreadyExists = errors.New("room already exists")
)

// Message represents a chat message.
type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "join", "leave", "system"
}

// Room represents a chat room.
type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UserCount int       `json:"user_count"`
}

// User represents a connected user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	RoomID   string `json:"room_id"`
}

// CreateRoomRequest is the request for creating a new room.
type CreateRoomRequest struct {
	Name string `json:"name"`
}

// JoinRoomRequest is the request for joining a room.
type JoinRoomRequest struct {
	RoomID   string `json:"room_id"`
	Username string `json:"username"`
}

// SendMessageRequest is the request for sending a message.
type SendMessageRequest struct {
	Content string `json:"content"`
}

// ChatEvent is published when a chat event occurs.
type ChatEvent struct {
	Type    string  `json:"type"` // "message", "join", "leave"
	RoomID  string  `json:"room_id"`
	Message Message `json:"message"`
}

// ValidateUsername validates a username.
func ValidateUsername(username string) error {
	if username == "" {
		return ErrUsernameEmpty
	}
	if len(username) > MaxUsernameLength {
		return ErrUsernameTooLong
	}
	if !utf8.ValidString(username) {
		return ErrUsernameInvalid
	}
	return nil
}

// ValidateRoomName validates a room name.
func ValidateRoomName(name string) error {
	if name == "" {
		return ErrRoomNameEmpty
	}
	if len(name) > MaxRoomNameLength {
		return ErrRoomNameTooLong
	}
	if !utf8.ValidString(name) {
		return ErrRoomNameInvalid
	}
	return nil
}

// ValidateMessage validates a message content.
func ValidateMessage(content string) error {
	if content == "" {
		return ErrMessageEmpty
	}
	if len(content) > MaxMessageLength {
		return ErrMessageTooLong
	}
	if !utf8.ValidString(content) {
		return ErrMessageInvalid
	}
	return nil
}

// RoomStore provides thread-safe storage for rooms and messages.
type RoomStore struct {
	mu         sync.RWMutex
	rooms      map[string]*Room
	users      map[string]*User           // userID -> User
	messages   map[string][]Message       // roomID -> messages
	roomUsers  map[string]map[string]bool // roomID -> set of userIDs
	maxHistory int
}

// NewRoomStore creates a new room store.
func NewRoomStore(maxHistory int) *RoomStore {
	if maxHistory <= 0 {
		maxHistory = 100
	}
	return &RoomStore{
		rooms:      make(map[string]*Room),
		users:      make(map[string]*User),
		messages:   make(map[string][]Message),
		roomUsers:  make(map[string]map[string]bool),
		maxHistory: maxHistory,
	}
}

// CreateRoom creates a new chat room.
func (s *RoomStore) CreateRoom(id, name string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	room := &Room{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
		UserCount: 0,
	}
	s.rooms[id] = room
	s.messages[id] = make([]Message, 0)
	s.roomUsers[id] = make(map[string]bool)
	return room
}

// GetRoom returns a room by ID.
func (s *RoomStore) GetRoom(id string) (*Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, exists := s.rooms[id]
	if !exists {
		return nil, false
	}
	// Return copy with current user count
	copy := *room
	copy.UserCount = len(s.roomUsers[id])
	return &copy, true
}

// ListRooms returns all rooms.
func (s *RoomStore) ListRooms() []Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		copy := *room
		copy.UserCount = len(s.roomUsers[room.ID])
		result = append(result, copy)
	}
	return result
}

// JoinRoom adds a user to a room.
func (s *RoomStore) JoinRoom(roomID string, user *User) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rooms[roomID]; !exists {
		return false
	}

	user.RoomID = roomID
	s.users[user.ID] = user
	s.roomUsers[roomID][user.ID] = true
	return true
}

// LeaveRoom removes a user from a room.
func (s *RoomStore) LeaveRoom(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return
	}

	if user.RoomID != "" {
		delete(s.roomUsers[user.RoomID], userID)
	}
	delete(s.users, userID)
}

// GetUser returns a user by ID.
func (s *RoomStore) GetUser(userID string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[userID]
	if !exists {
		return nil, false
	}
	copy := *user
	return &copy, true
}

// GetRoomUsers returns all users in a room.
func (s *RoomStore) GetRoomUsers(roomID string) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userIDs, exists := s.roomUsers[roomID]
	if !exists {
		return nil
	}

	result := make([]User, 0, len(userIDs))
	for userID := range userIDs {
		if user, ok := s.users[userID]; ok {
			result = append(result, *user)
		}
	}
	return result
}

// AddMessage adds a message to a room's history.
func (s *RoomStore) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, exists := s.messages[msg.RoomID]
	if !exists {
		return
	}

	messages = append(messages, msg)
	// Trim to max history
	if len(messages) > s.maxHistory {
		messages = messages[len(messages)-s.maxHistory:]
	}
	s.messages[msg.RoomID] = messages
}

// GetHistory returns message history for a room.
func (s *RoomStore) GetHistory(roomID string, limit int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages, exists := s.messages[roomID]
	if !exists {
		return nil
	}

	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}

	start := len(messages) - limit
	result := make([]Message, limit)
	copy(result, messages[start:])
	return result
}
