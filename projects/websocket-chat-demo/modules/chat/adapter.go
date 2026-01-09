package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ServiceAdapter provides a type-safe interface for calling chat services.
// It wraps the ServiceContainer and handles serialization/deserialization.
type ServiceAdapter struct {
	container mono.ServiceContainer
}

// NewServiceAdapter creates a new chat service adapter.
func NewServiceAdapter(container mono.ServiceContainer) *ServiceAdapter {
	return &ServiceAdapter{container: container}
}

// CreateRoom creates a new chat room.
func (a *ServiceAdapter) CreateRoom(ctx context.Context, name string) (*Room, error) {
	req := CreateRoomServiceRequest{Name: name}
	var resp CreateRoomServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceCreateRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to call create-room service: %w", err)
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Room, nil
}

// GetRoom returns a room by ID.
func (a *ServiceAdapter) GetRoom(ctx context.Context, roomID string) (*Room, bool, error) {
	req := GetRoomServiceRequest{RoomID: roomID}
	var resp GetRoomServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, false, fmt.Errorf("failed to call get-room service: %w", err)
	}

	if resp.Error != "" {
		return nil, false, errors.New(resp.Error)
	}
	return resp.Room, resp.Exists, nil
}

// ListRooms returns all active rooms.
func (a *ServiceAdapter) ListRooms(ctx context.Context) ([]Room, error) {
	var req struct{} // empty request
	var resp ListRoomsServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceListRooms,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to call list-rooms service: %w", err)
	}

	return resp.Rooms, nil
}

// JoinRoom adds a user to a room.
func (a *ServiceAdapter) JoinRoom(ctx context.Context, roomID, userID, username string) (*User, error) {
	req := JoinRoomServiceRequest{
		RoomID:   roomID,
		UserID:   userID,
		Username: username,
	}
	var resp JoinRoomServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceJoinRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to call join-room service: %w", err)
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.User, nil
}

// LeaveRoom removes a user from a room.
func (a *ServiceAdapter) LeaveRoom(ctx context.Context, userID string) error {
	req := LeaveRoomServiceRequest{UserID: userID}
	var resp LeaveRoomServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceLeaveRoom,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("failed to call leave-room service: %w", err)
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

// SendMessage sends a message to a room.
func (a *ServiceAdapter) SendMessage(ctx context.Context, userID, content string) error {
	req := SendMessageServiceRequest{
		UserID:  userID,
		Content: content,
	}
	var resp SendMessageServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceSendMessage,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return fmt.Errorf("failed to call send-message service: %w", err)
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

// GetUser returns a user by ID.
func (a *ServiceAdapter) GetUser(ctx context.Context, userID string) (*User, bool, error) {
	req := GetUserServiceRequest{UserID: userID}
	var resp GetUserServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetUser,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, false, fmt.Errorf("failed to call get-user service: %w", err)
	}

	if resp.Error != "" {
		return nil, false, errors.New(resp.Error)
	}
	return resp.User, resp.Exists, nil
}

// GetHistory returns message history for a room.
func (a *ServiceAdapter) GetHistory(ctx context.Context, roomID string, limit int) ([]Message, error) {
	req := GetHistoryServiceRequest{
		RoomID: roomID,
		Limit:  limit,
	}
	var resp GetHistoryServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetHistory,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to call get-history service: %w", err)
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Messages, nil
}

// GetRoomUsers returns all users in a room.
func (a *ServiceAdapter) GetRoomUsers(ctx context.Context, roomID string) ([]User, error) {
	req := GetRoomUsersServiceRequest{RoomID: roomID}
	var resp GetRoomUsersServiceResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		ServiceGetRoomUsers,
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("failed to call get-room-users service: %w", err)
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Users, nil
}
