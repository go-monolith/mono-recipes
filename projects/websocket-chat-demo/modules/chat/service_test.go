package chat

import (
	"context"
	"testing"
)

func TestService_CreateRoom(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	tests := []struct {
		name        string
		roomName    string
		createdBy   string
		expectError bool
	}{
		{
			name:        "valid room creation",
			roomName:    "General",
			createdBy:   "user1",
			expectError: false,
		},
		{
			name:        "another valid room",
			roomName:    "Random",
			createdBy:   "user2",
			expectError: false,
		},
		{
			name:        "empty room name",
			roomName:    "",
			createdBy:   "user1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, err := service.CreateRoom(ctx, tt.roomName, tt.createdBy)

			if tt.expectError {
				if err == nil {
					t.Error("CreateRoom() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("CreateRoom() unexpected error: %v", err)
			}

			if room == nil {
				t.Fatal("CreateRoom() returned nil room")
			}

			if room.Name != tt.roomName {
				t.Errorf("CreateRoom() room.Name = %q, want %q", room.Name, tt.roomName)
			}

			if room.ID == "" {
				t.Error("CreateRoom() room.ID should not be empty")
			}

			if room.CreatedAt.IsZero() {
				t.Error("CreateRoom() room.CreatedAt should not be zero")
			}
		})
	}
}

func TestService_GetRoom(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	// Create a room first
	room, err := service.CreateRoom(ctx, "TestRoom", "user1")
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	tests := []struct {
		name        string
		roomID      string
		expectError bool
	}{
		{
			name:        "existing room",
			roomID:      room.ID,
			expectError: false,
		},
		{
			name:        "non-existent room",
			roomID:      "non-existent-id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetRoom(ctx, tt.roomID)

			if tt.expectError {
				if err == nil {
					t.Error("GetRoom() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetRoom() unexpected error: %v", err)
			}

			if result.ID != room.ID {
				t.Errorf("GetRoom() room.ID = %q, want %q", result.ID, room.ID)
			}
		})
	}
}

func TestService_ListRooms(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	// Initially empty
	rooms := service.ListRooms(ctx)
	if len(rooms) != 0 {
		t.Errorf("ListRooms() initial count = %d, want 0", len(rooms))
	}

	// Create some rooms
	_, _ = service.CreateRoom(ctx, "Room1", "user1")
	_, _ = service.CreateRoom(ctx, "Room2", "user2")
	_, _ = service.CreateRoom(ctx, "Room3", "user3")

	rooms = service.ListRooms(ctx)
	if len(rooms) != 3 {
		t.Errorf("ListRooms() count = %d, want 3", len(rooms))
	}
}

func TestService_JoinRoom(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")

	tests := []struct {
		name        string
		roomID      string
		userID      string
		username    string
		expectError bool
	}{
		{
			name:        "join existing room",
			roomID:      room.ID,
			userID:      "user1",
			username:    "User1",
			expectError: false,
		},
		{
			name:        "join same room again",
			roomID:      room.ID,
			userID:      "user1",
			username:    "User1",
			expectError: false,
		},
		{
			name:        "join non-existent room",
			roomID:      "non-existent",
			userID:      "user1",
			username:    "User1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.JoinRoom(ctx, tt.roomID, tt.userID, tt.username)

			if tt.expectError {
				if err == nil {
					t.Error("JoinRoom() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("JoinRoom() unexpected error: %v", err)
			}
		})
	}
}

func TestService_LeaveRoom(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")
	_ = service.JoinRoom(ctx, room.ID, "user1", "User1")

	tests := []struct {
		name        string
		roomID      string
		userID      string
		expectError bool
	}{
		{
			name:        "leave existing room",
			roomID:      room.ID,
			userID:      "user1",
			expectError: false,
		},
		{
			name:        "leave non-existent room",
			roomID:      "non-existent",
			userID:      "user1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.LeaveRoom(ctx, tt.roomID, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("LeaveRoom() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("LeaveRoom() unexpected error: %v", err)
			}
		})
	}
}

func TestService_SendMessage(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")

	tests := []struct {
		name        string
		roomID      string
		userID      string
		username    string
		content     string
		expectError bool
	}{
		{
			name:        "valid message",
			roomID:      room.ID,
			userID:      "user1",
			username:    "User1",
			content:     "Hello, World!",
			expectError: false,
		},
		{
			name:        "empty message",
			roomID:      room.ID,
			userID:      "user1",
			username:    "User1",
			content:     "",
			expectError: true,
		},
		{
			name:        "message to non-existent room",
			roomID:      "non-existent",
			userID:      "user1",
			username:    "User1",
			content:     "Hello",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := service.SendMessage(ctx, tt.roomID, tt.userID, tt.username, tt.content)

			if tt.expectError {
				if err == nil {
					t.Error("SendMessage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("SendMessage() unexpected error: %v", err)
			}

			if msg == nil {
				t.Fatal("SendMessage() returned nil message")
			}

			if msg.Content != tt.content {
				t.Errorf("SendMessage() message.Content = %q, want %q", msg.Content, tt.content)
			}

			if msg.UserID != tt.userID {
				t.Errorf("SendMessage() message.UserID = %q, want %q", msg.UserID, tt.userID)
			}

			if msg.ID == "" {
				t.Error("SendMessage() message.ID should not be empty")
			}
		})
	}
}

func TestService_GetHistory(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")

	// Send some messages
	for i := 0; i < 5; i++ {
		_, _ = service.SendMessage(ctx, room.ID, "user1", "User1", "Message")
	}

	tests := []struct {
		name          string
		roomID        string
		limit         int
		expectedCount int
		expectError   bool
	}{
		{
			name:          "get all messages",
			roomID:        room.ID,
			limit:         0,
			expectedCount: 5,
			expectError:   false,
		},
		{
			name:          "get limited messages",
			roomID:        room.ID,
			limit:         3,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "non-existent room",
			roomID:        "non-existent",
			limit:         10,
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := service.GetHistory(ctx, tt.roomID, tt.limit)

			if tt.expectError {
				if err == nil {
					t.Error("GetHistory() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetHistory() unexpected error: %v", err)
			}

			if len(messages) != tt.expectedCount {
				t.Errorf("GetHistory() count = %d, want %d", len(messages), tt.expectedCount)
			}
		})
	}
}

func TestService_GetHistory_MaxSize(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")

	// Send more messages than maxHistorySize
	for i := 0; i < maxHistorySize+50; i++ {
		_, _ = service.SendMessage(ctx, room.ID, "user1", "User1", "Message")
	}

	messages, err := service.GetHistory(ctx, room.ID, 0)
	if err != nil {
		t.Fatalf("GetHistory() error: %v", err)
	}

	if len(messages) != maxHistorySize {
		t.Errorf("GetHistory() count = %d, want %d (maxHistorySize)", len(messages), maxHistorySize)
	}
}

func TestService_GetRoomMembers(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")

	// Initially empty
	members, err := service.GetRoomMembers(ctx, room.ID)
	if err != nil {
		t.Fatalf("GetRoomMembers() error: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("GetRoomMembers() initial count = %d, want 0", len(members))
	}

	// Add some users
	_ = service.JoinRoom(ctx, room.ID, "user1", "User1")
	_ = service.JoinRoom(ctx, room.ID, "user2", "User2")
	_ = service.JoinRoom(ctx, room.ID, "user3", "User3")

	members, err = service.GetRoomMembers(ctx, room.ID)
	if err != nil {
		t.Fatalf("GetRoomMembers() error: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("GetRoomMembers() count = %d, want 3", len(members))
	}

	// Test non-existent room
	_, err = service.GetRoomMembers(ctx, "non-existent")
	if err == nil {
		t.Error("GetRoomMembers() expected error for non-existent room")
	}
}

func TestService_RoomExists(t *testing.T) {
	ctx := context.Background()
	service := NewService()

	room, _ := service.CreateRoom(ctx, "TestRoom", "owner")

	if !service.RoomExists(ctx, room.ID) {
		t.Error("RoomExists() = false, want true for existing room")
	}

	if service.RoomExists(ctx, "non-existent") {
		t.Error("RoomExists() = true, want false for non-existent room")
	}
}

func BenchmarkService_SendMessage(b *testing.B) {
	ctx := context.Background()
	service := NewService()
	room, _ := service.CreateRoom(ctx, "BenchRoom", "owner")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.SendMessage(ctx, room.ID, "user1", "User1", "Benchmark message")
	}
}

func BenchmarkService_GetHistory(b *testing.B) {
	ctx := context.Background()
	service := NewService()
	room, _ := service.CreateRoom(ctx, "BenchRoom", "owner")

	// Populate with messages
	for i := 0; i < 100; i++ {
		_, _ = service.SendMessage(ctx, room.ID, "user1", "User1", "Message")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetHistory(ctx, room.ID, 50)
	}
}
