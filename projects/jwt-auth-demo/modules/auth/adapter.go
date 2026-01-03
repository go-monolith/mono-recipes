package auth

import (
	"context"
	"encoding/json"
	"fmt"

	domain "github.com/example/jwt-auth-demo/domain/user"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// AuthPort defines the interface for authentication operations.
// This is the port that other modules use to access auth functionality.
type AuthPort interface {
	ValidateToken(ctx context.Context, token string) (*domain.Claims, error)
	GetUser(ctx context.Context, userID string) (*domain.User, error)
}

// AuthAdapter implements AuthPort using the service container.
type AuthAdapter struct {
	container mono.ServiceContainer
}

// NewAuthAdapter creates a new AuthAdapter.
func NewAuthAdapter(container mono.ServiceContainer) *AuthAdapter {
	return &AuthAdapter{
		container: container,
	}
}

// ValidateToken validates an access token and returns claims.
func (a *AuthAdapter) ValidateToken(ctx context.Context, token string) (*domain.Claims, error) {
	req := ValidateTokenRequest{Token: token}
	var resp ValidateTokenResponse

	if err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"validate-token",
		json.Marshal,
		json.Unmarshal,
		&req,
		&resp,
	); err != nil {
		return nil, fmt.Errorf("validate-token request failed: %w", err)
	}

	if !resp.Valid {
		return nil, fmt.Errorf("token validation failed: %s", resp.Error)
	}

	return &domain.Claims{
		UserID: resp.UserID,
		Email:  resp.Email,
	}, nil
}

// GetUser retrieves a user by ID.
func (a *AuthAdapter) GetUser(ctx context.Context, userID string) (*domain.User, error) {
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
		return nil, fmt.Errorf("get-user request failed: %w", err)
	}

	return &domain.User{
		ID:        resp.ID,
		Email:     resp.Email,
		CreatedAt: resp.CreatedAt,
	}, nil
}
