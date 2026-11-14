package api

import (
	"context"

	"github.com/go-monolith/mono"

	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

// authAdapter is the driven adapter behind AuthPort: each method is one call
// of an auth contract through the container received in
// SetDependencyServiceContainer. Errors returned by the auth handlers keep
// their message, wrapped with the contract identity.
type authAdapter struct {
	container mono.ServiceContainer
}

var _ AuthPort = authAdapter{}

func (a authAdapter) Register(ctx context.Context, req authcontract.RegisterRequest) (authcontract.RegisterResponse, error) {
	return authcontract.Register.Call(ctx, a.container, req)
}

func (a authAdapter) Login(ctx context.Context, req authcontract.LoginRequest) (authcontract.TokenResponse, error) {
	return authcontract.Login.Call(ctx, a.container, req)
}

func (a authAdapter) RefreshToken(ctx context.Context, req authcontract.RefreshRequest) (authcontract.TokenResponse, error) {
	return authcontract.RefreshToken.Call(ctx, a.container, req)
}

func (a authAdapter) ValidateToken(ctx context.Context, req authcontract.ValidateTokenRequest) (authcontract.ValidateTokenResponse, error) {
	return authcontract.ValidateToken.Call(ctx, a.container, req)
}

func (a authAdapter) GetUser(ctx context.Context, req authcontract.GetUserRequest) (authcontract.GetUserResponse, error) {
	return authcontract.GetUser.Call(ctx, a.container, req)
}
