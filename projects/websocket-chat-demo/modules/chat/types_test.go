package chat

import (
	"testing"
	"time"
)

func TestRoomStore_CreateRoom(t *testing.T) {
	store := NewRoomStore(100)

	room := store.CreateRoom("test-id", "Test Room")

	if room.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %q", room.ID)
	}
	if room.Name != "Test Room" {
		t.Errorf("Expected Name 'Test Room', got %q", room.Name)
	}
	if room.UserCount != 0 {
		t.Errorf("Expected UserCount 0, got %d", room.UserCount)
	}
}

func TestRoomStore_GetRoom(t *testing.T) {
	store := NewRoomStore(100)
	store.CreateRoom("test-id", "Test Room")

	room, exists := store.GetRoom("test-id")
	if !exists {
		t.Fatal("Expected room to exist")
	}
	if room.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %q", room.ID)
	}

	_, exists = store.GetRoom("nonexistent")
	if exists {
		t.Error("Expected nonexistent room to not exist")
	}
}

func TestRoomStore_ListRooms(t *testing.T) {
	store := NewRoomStore(100)
	store.CreateRoom("room1", "Room 1")
	store.CreateRoom("room2", "Room 2")

	rooms := store.ListRooms()
	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}
}

func TestRoomStore_JoinAndLeaveRoom(t *testing.T) {
	store := NewRoomStore(100)
	store.CreateRoom("room1", "Room 1")

	user := &User{ID: "user1", Username: "Alice"}
	joined := store.JoinRoom("room1", user)
	if !joined {
		t.Fatal("Expected user to join room")
	}

	room, _ := store.GetRoom("room1")
	if room.UserCount != 1 {
		t.Errorf("Expected UserCount 1, got %d", room.UserCount)
	}

	retrievedUser, exists := store.GetUser("user1")
	if !exists {
		t.Fatal("Expected user to exist")
	}
	if retrievedUser.RoomID != "room1" {
		t.Errorf("Expected RoomID 'room1', got %q", retrievedUser.RoomID)
	}

	store.LeaveRoom("user1")
	room, _ = store.GetRoom("room1")
	if room.UserCount != 0 {
		t.Errorf("Expected UserCount 0 after leave, got %d", room.UserCount)
	}
}

func TestRoomStore_JoinNonexistentRoom(t *testing.T) {
	store := NewRoomStore(100)

	user := &User{ID: "user1", Username: "Alice"}
	joined := store.JoinRoom("nonexistent", user)
	if joined {
		t.Error("Expected user to not join nonexistent room")
	}
}

func TestRoomStore_GetRoomUsers(t *testing.T) {
	store := NewRoomStore(100)
	store.CreateRoom("room1", "Room 1")

	store.JoinRoom("room1", &User{ID: "user1", Username: "Alice"})
	store.JoinRoom("room1", &User{ID: "user2", Username: "Bob"})

	users := store.GetRoomUsers("room1")
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestRoomStore_AddAndGetHistory(t *testing.T) {
	store := NewRoomStore(5) // Small history for testing
	store.CreateRoom("room1", "Room 1")

	// Add messages
	for i := range 10 {
		store.AddMessage(Message{
			ID:        string(rune('a' + i)),
			RoomID:    "room1",
			Content:   "Message",
			Timestamp: time.Now(),
		})
	}

	// Should only keep last 5
	history := store.GetHistory("room1", 10)
	if len(history) != 5 {
		t.Errorf("Expected 5 messages (max history), got %d", len(history))
	}
}

func TestRoomStore_GetHistoryWithLimit(t *testing.T) {
	store := NewRoomStore(100)
	store.CreateRoom("room1", "Room 1")

	for i := range 10 {
		store.AddMessage(Message{
			ID:        string(rune('a' + i)),
			RoomID:    "room1",
			Content:   "Message",
			Timestamp: time.Now(),
		})
	}

	// Get only last 3
	history := store.GetHistory("room1", 3)
	if len(history) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(history))
	}
}

func TestRoomStore_HistoryNonexistentRoom(t *testing.T) {
	store := NewRoomStore(100)

	history := store.GetHistory("nonexistent", 10)
	if history != nil {
		t.Error("Expected nil history for nonexistent room")
	}
}
