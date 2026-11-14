package auth_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-monolith/mono/monotest"

	"github.com/example/jwt-auth-demo/modules/auth"
	"github.com/example/jwt-auth-demo/modules/auth/contract"
)

// newModule creates an auth module backed by a fresh SQLite file.
func newModule(t *testing.T) *auth.AuthModule {
	t.Helper()
	t.Setenv("JWT_AUTH_DB_PATH", filepath.Join(t.TempDir(), "auth.db"))
	return auth.NewModule()
}

// TestAuthModuleConformance checks the auth module against the Ports and
// Adapters rules and calls every contract it provides, in the order a client
// would: register, login, validate, get-user, refresh.
func TestAuthModuleConformance(t *testing.T) {
	monotest.CheckModule(t, newModule(t),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			c := h.Container()
			user, err := contract.Register.Call(ctx, c, contract.RegisterRequest{Email: "alice@example.com", Password: "password123"})
			if err != nil {
				return err
			}
			tokens, err := contract.Login.Call(ctx, c, contract.LoginRequest{Email: "alice@example.com", Password: "password123"})
			if err != nil {
				return err
			}
			if tokens.TokenType != "Bearer" || tokens.AccessToken == "" || tokens.RefreshToken == "" {
				return fmt.Errorf("login tokens = %+v", tokens)
			}
			claims, err := contract.ValidateToken.Call(ctx, c, contract.ValidateTokenRequest{Token: tokens.AccessToken})
			if err != nil {
				return err
			}
			if !claims.Valid || claims.UserID != user.ID {
				return fmt.Errorf("validate-token = %+v, want valid for %s", claims, user.ID)
			}
			profile, err := contract.GetUser.Call(ctx, c, contract.GetUserRequest{UserID: user.ID})
			if err != nil {
				return err
			}
			if profile.Email != "alice@example.com" {
				return fmt.Errorf("get-user email = %q", profile.Email)
			}
			refreshed, err := contract.RefreshToken.Call(ctx, c, contract.RefreshRequest{RefreshToken: tokens.RefreshToken})
			if err != nil {
				return err
			}
			if refreshed.AccessToken == "" {
				return fmt.Errorf("refresh-token returned no access token")
			}
			return nil
		}))
}

// TestAuthErrorsCrossTheContract checks that domain errors reach the caller
// with their message (the api module maps them to HTTP status codes), and that
// an invalid token is a response, not an error.
func TestAuthErrorsCrossTheContract(t *testing.T) {
	ctx := context.Background()
	h := monotest.New(t, newModule(t))
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()
	c := h.Container()

	req := contract.RegisterRequest{Email: "bob@example.com", Password: "password123"}
	if _, err := contract.Register.Call(ctx, c, req); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"duplicate email", func() error { _, err := contract.Register.Call(ctx, c, req); return err }, auth.ErrUserExists.Error()},
		{"weak password", func() error {
			_, err := contract.Register.Call(ctx, c, contract.RegisterRequest{Email: "carol@example.com", Password: "short"})
			return err
		}, auth.ErrWeakPassword.Error()},
		{"wrong password", func() error {
			_, err := contract.Login.Call(ctx, c, contract.LoginRequest{Email: req.Email, Password: "wrong-password"})
			return err
		}, auth.ErrInvalidCredentials.Error()},
	}
	for _, tc := range cases {
		err := tc.call()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to contain %q", tc.name, err, tc.want)
		}
	}

	resp, err := contract.ValidateToken.Call(ctx, c, contract.ValidateTokenRequest{Token: "not-a-jwt"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Valid || resp.Error == "" {
		t.Errorf("validate-token(garbage) = %+v, want invalid with an error message", resp)
	}
}
