package api

import (
	"context"

	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

// AuthPort is the driven port the HTTP layer needs from the auth module. The
// api module owns it; it speaks the auth contract's DTOs and says nothing
// about NATS, so handlers and middleware can be tested against a fake.
type AuthPort interface {
	Register(ctx context.Context, req authcontract.RegisterRequest) (authcontract.RegisterResponse, error)
	Login(ctx context.Context, req authcontract.LoginRequest) (authcontract.TokenResponse, error)
	RefreshToken(ctx context.Context, req authcontract.RefreshRequest) (authcontract.TokenResponse, error)
	ValidateToken(ctx context.Context, req authcontract.ValidateTokenRequest) (authcontract.ValidateTokenResponse, error)
	GetUser(ctx context.Context, req authcontract.GetUserRequest) (authcontract.GetUserResponse, error)
}
