package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// UserPort defines the interface for user operations (used by other modules).
// This is the "port" in hexagonal architecture.
type UserPort interface {
	GetUser(ctx context.Context, userID string) (*UserInfo, error)
	ValidateUser(ctx context.Context, userID string) (bool, error)
}

// userAdapter wraps ServiceContainer for type-safe cross-module communication.
// This is the "adapter" that implements the port interface.
type userAdapter struct {
	container mono.ServiceContainer
}

// NewUserAdapter creates a new adapter for user services.
// container is the ServiceContainer from the user module received via SetDependencyServiceContainer.
func NewUserAdapter(container mono.ServiceContainer) UserPort {
	if container == nil {
		panic("user adapter requires non-nil ServiceContainer")
	}
	return &userAdapter{container: container}
}

// GetUser retrieves user information by ID via the get-user service.
func (a *userAdapter) GetUser(ctx context.Context, userID string) (*UserInfo, error) {
	req := GetUserRequest{UserID: userID}
	var resp GetUserResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"get-user",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("get-user service call failed: %w", err)
	}

	if !resp.Found {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return resp.User, nil
}

// ValidateUser checks if a user exists via the validate-user service.
func (a *userAdapter) ValidateUser(ctx context.Context, userID string) (bool, error) {
	req := ValidateUserRequest{UserID: userID}
	var resp ValidateUserResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"validate-user",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return false, fmt.Errorf("validate-user service call failed: %w", err)
	}

	return resp.Valid, nil
}
