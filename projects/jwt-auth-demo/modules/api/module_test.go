package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/jwt-auth-demo/modules/api"
	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

const (
	stubUserID = "user-123"
	stubEmail  = "alice@example.com"
	stubToken  = "access-token"
)

// standInAuth is a NATS-free stand-in for the auth module, registered through
// the auth module's own contracts. It accepts one access token and rejects
// duplicate registration of stubEmail with the auth module's error message.
func standInAuth(t *testing.T, bus *monotest.EventBus) *monotest.Container {
	t.Helper()
	c := monotest.NewContainer("auth", bus)
	now := time.Now()
	tokens := authcontract.TokenResponse{AccessToken: stubToken, RefreshToken: "refresh-token", ExpiresIn: 900, TokenType: "Bearer"}
	registered := false

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(authcontract.Register.Register(c, func(_ context.Context, req authcontract.RegisterRequest, _ *types.Msg) (authcontract.RegisterResponse, error) {
		if registered {
			return authcontract.RegisterResponse{}, errors.New("user with this email already exists")
		}
		registered = true
		return authcontract.RegisterResponse{ID: stubUserID, Email: req.Email, CreatedAt: now}, nil
	}))
	must(authcontract.Login.Register(c, func(context.Context, authcontract.LoginRequest, *types.Msg) (authcontract.TokenResponse, error) {
		return tokens, nil
	}))
	must(authcontract.RefreshToken.Register(c, func(context.Context, authcontract.RefreshRequest, *types.Msg) (authcontract.TokenResponse, error) {
		return tokens, nil
	}))
	must(authcontract.ValidateToken.Register(c, func(_ context.Context, req authcontract.ValidateTokenRequest, _ *types.Msg) (authcontract.ValidateTokenResponse, error) {
		if req.Token != stubToken {
			return authcontract.ValidateTokenResponse{Valid: false, Error: "invalid token"}, nil
		}
		return authcontract.ValidateTokenResponse{Valid: true, UserID: stubUserID, Email: stubEmail}, nil
	}))
	must(authcontract.GetUser.Register(c, func(_ context.Context, req authcontract.GetUserRequest, _ *types.Msg) (authcontract.GetUserResponse, error) {
		return authcontract.GetUserResponse{ID: req.UserID, Email: stubEmail, CreatedAt: now}, nil
	}))
	return c
}

// do sends an HTTP request to the module's Fiber app and checks the status.
func do(module *api.APIModule, method, path, bearer string, body any, wantStatus int, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := module.App().Test(req, 5000)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		return fmt.Errorf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, wantStatus, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// TestAPIModuleConformance checks the api module against the Ports and
// Adapters rules. The exercise hits every route, so every consumed auth
// contract is called through the adapter (validate-token via the middleware).
func TestAPIModuleConformance(t *testing.T) {
	bus := monotest.NewEventBus()
	module := api.NewModuleWithAddr("127.0.0.1:0")
	monotest.CheckModule(t, module,
		monotest.WithDependency("auth", standInAuth(t, bus)),
		monotest.WithExercise(func(context.Context, *monotest.Harness) error {
			creds := api.RegisterRequest{Email: stubEmail, Password: "password123"}
			if err := do(module, http.MethodPost, "/api/v1/auth/register", "", creds, http.StatusCreated, nil); err != nil {
				return err
			}
			var tokens api.TokenResponse
			if err := do(module, http.MethodPost, "/api/v1/auth/login", "", api.LoginRequest(creds), http.StatusOK, &tokens); err != nil {
				return err
			}
			if err := do(module, http.MethodPost, "/api/v1/auth/refresh", "", api.RefreshRequest{RefreshToken: tokens.RefreshToken}, http.StatusOK, nil); err != nil {
				return err
			}
			var profile api.ProfileResponse
			if err := do(module, http.MethodGet, "/api/v1/profile", tokens.AccessToken, nil, http.StatusOK, &profile); err != nil {
				return err
			}
			if profile.ID != stubUserID || profile.Email != stubEmail {
				return fmt.Errorf("profile = %+v", profile)
			}
			return nil
		}))
}

// TestAPIErrorMapping checks how auth contract results map to HTTP statuses.
func TestAPIErrorMapping(t *testing.T) {
	ctx := context.Background()
	bus := monotest.NewEventBus()
	module := api.NewModuleWithAddr("127.0.0.1:0")
	h := monotest.New(t, module, monotest.WithDependency("auth", standInAuth(t, bus)))
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	creds := api.RegisterRequest{Email: stubEmail, Password: "password123"}
	if err := do(module, http.MethodPost, "/api/v1/auth/register", "", creds, http.StatusCreated, nil); err != nil {
		t.Fatal(err)
	}
	// The auth handler's error message survives the contract call.
	if err := do(module, http.MethodPost, "/api/v1/auth/register", "", creds, http.StatusConflict, nil); err != nil {
		t.Error(err)
	}
	// validate-token answers Valid=false: the middleware rejects the request.
	if err := do(module, http.MethodGet, "/api/v1/profile", "forged-token", nil, http.StatusUnauthorized, nil); err != nil {
		t.Error(err)
	}
	if err := do(module, http.MethodGet, "/api/v1/profile", "", nil, http.StatusUnauthorized, nil); err != nil {
		t.Error(err)
	}
}
