// Package contract is the published port of the auth module: the DTOs of its
// services and the typed contracts other modules call them through. It is the
// only package of the auth module another module may import.
//
// A contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse represents a user registration response.
type RegisterResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest represents a user login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse is the access/refresh token pair returned by login and
// refresh-token.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ValidateTokenRequest represents a token validation request.
type ValidateTokenRequest struct {
	Token string `json:"token"`
}

// ValidateTokenResponse represents a token validation response. An invalid or
// expired token is reported with Valid=false and Error set, not as an error.
type ValidateTokenResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
	Error  string `json:"error,omitempty"`
}

// GetUserRequest represents a get user request.
type GetUserRequest struct {
	UserID string `json:"user_id"`
}

// GetUserResponse represents a get user response.
type GetUserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Register creates a user account. Subject: services.auth.register
var Register = helper.RequestReplyService[RegisterRequest, RegisterResponse]("auth", "register")

// Login authenticates a user and returns tokens. Subject: services.auth.login
var Login = helper.RequestReplyService[LoginRequest, TokenResponse]("auth", "login")

// RefreshToken exchanges a refresh token for a new token pair.
// Subject: services.auth.refresh-token
var RefreshToken = helper.RequestReplyService[RefreshRequest, TokenResponse]("auth", "refresh-token")

// ValidateToken validates an access token. Subject: services.auth.validate-token
var ValidateToken = helper.RequestReplyService[ValidateTokenRequest, ValidateTokenResponse]("auth", "validate-token")

// GetUser returns a user by ID. Subject: services.auth.get-user
var GetUser = helper.RequestReplyService[GetUserRequest, GetUserResponse]("auth", "get-user")
