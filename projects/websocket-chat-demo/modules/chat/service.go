package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	domain "github.com/example/websocket-chat-demo/domain/chat"
	"github.com/google/uuid"
)

// maxHistorySize is the maximum number of messages to keep per room.
const maxHistorySize = 100

// Service provides chat room operations.
type Service struct {
	rooms    map[string]*domain.Room
	messages map[string][]*domain.Message // roomID -> messages
	members  map[string]map[string]*domain.User // roomID -> userID -> user
	mu       sync.RWMutex
}

// NewService creates a new chat service.
func NewService() *Service {
	return &Service{
		rooms:    make(map[string]*domain.Room),
		messages: make(map[string][]*domain.Message),
		members:  make(map[string]map[string]*domain.User),
	}
}

// CreateRoom creates a new chat room.
func (s *Service) CreateRoom(_ context.Context, name, createdBy string) (*domain.Room, error) {
	if name == "" {
		return nil, fmt.Errorf("room name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	room := &domain.Room{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now(),
	}

	s.rooms[room.ID] = room
	s.messages[room.ID] = make([]*domain.Message, 0)
	s.members[room.ID] = make(map[string]*domain.User)

	return room, nil
}

// GetRoom retrieves a room by ID.
func (s *Service) GetRoom(_ context.Context, roomID string) (*domain.Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	return room, nil
}

// ListRooms returns all available rooms.
func (s *Service) ListRooms(_ context.Context) []*domain.Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]*domain.Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}

// JoinRoom adds a user to a room.
func (s *Service) JoinRoom(_ context.Context, roomID, userID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rooms[roomID]; !ok {
		return fmt.Errorf("room not found: %s", roomID)
	}

	if s.members[roomID] == nil {
		s.members[roomID] = make(map[string]*domain.User)
	}

	s.members[roomID][userID] = &domain.User{
		ID:       userID,
		Username: username,
		RoomID:   roomID,
	}

	return nil
}

// LeaveRoom removes a user from a room.
func (s *Service) LeaveRoom(_ context.Context, roomID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rooms[roomID]; !ok {
		return fmt.Errorf("room not found: %s", roomID)
	}

	delete(s.members[roomID], userID)
	return nil
}

// SendMessage stores a message in a room.
func (s *Service) SendMessage(_ context.Context, roomID, userID, username, content string) (*domain.Message, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rooms[roomID]; !ok {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	msg := &domain.Message{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Append message and enforce max size
	s.messages[roomID] = append(s.messages[roomID], msg)
	if len(s.messages[roomID]) > maxHistorySize {
		s.messages[roomID] = s.messages[roomID][len(s.messages[roomID])-maxHistorySize:]
	}

	return msg, nil
}

// GetHistory retrieves message history for a room.
func (s *Service) GetHistory(_ context.Context, roomID string, limit int) ([]*domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.rooms[roomID]; !ok {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	messages := s.messages[roomID]
	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}

	// Return the last 'limit' messages
	start := len(messages) - limit
	if start < 0 {
		start = 0
	}

	// Calculate actual length from the range
	resultSlice := messages[start:]
	result := make([]*domain.Message, len(resultSlice))
	copy(result, resultSlice)
	return result, nil
}

// GetRoomMembers returns all members in a room.
func (s *Service) GetRoomMembers(_ context.Context, roomID string) ([]*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.rooms[roomID]; !ok {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	members := make([]*domain.User, 0, len(s.members[roomID]))
	for _, user := range s.members[roomID] {
		members = append(members, user)
	}

	return members, nil
}

// RoomExists checks if a room exists.
func (s *Service) RoomExists(_ context.Context, roomID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.rooms[roomID]
	return ok
}
