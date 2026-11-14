package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

// mockAuthPort is a fake AuthPort: the middleware only calls ValidateToken,
// so the other methods come from the embedded (nil) interface and would panic
// if called.
type mockAuthPort struct {
	AuthPort
	validateTokenFunc func(ctx context.Context, token string) (authcontract.ValidateTokenResponse, error)
}

func (m *mockAuthPort) ValidateToken(ctx context.Context, req authcontract.ValidateTokenRequest) (authcontract.ValidateTokenResponse, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, req.Token)
	}
	return authcontract.ValidateTokenResponse{}, errors.New("not implemented")
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		mockAuth       *mockAuthPort
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			mockAuth:       &mockAuthPort{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `"Authorization header is required"`,
		},
		{
			name:           "invalid authorization format - no bearer",
			authHeader:     "Basic token123",
			mockAuth:       &mockAuthPort{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `Invalid authorization header format`,
		},
		{
			name:           "bearer without token",
			authHeader:     "Bearer ",
			mockAuth:       &mockAuthPort{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `unauthorized`, // Fiber trims trailing spaces, so "Bearer " becomes "Bearer" which fails prefix check
		},
		{
			name:       "transport error",
			authHeader: "Bearer some-token",
			mockAuth: &mockAuthPort{
				validateTokenFunc: func(ctx context.Context, token string) (authcontract.ValidateTokenResponse, error) {
					return authcontract.ValidateTokenResponse{}, errors.New("service unavailable")
				},
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `"Invalid or expired token"`,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			mockAuth: &mockAuthPort{
				validateTokenFunc: func(ctx context.Context, token string) (authcontract.ValidateTokenResponse, error) {
					return authcontract.ValidateTokenResponse{Valid: false, Error: "invalid token"}, nil
				},
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `"Invalid or expired token"`,
		},
		{
			name:       "valid token",
			authHeader: "Bearer valid-token",
			mockAuth: &mockAuthPort{
				validateTokenFunc: func(ctx context.Context, token string) (authcontract.ValidateTokenResponse, error) {
					return authcontract.ValidateTokenResponse{
						Valid:  true,
						UserID: "user-123",
						Email:  "test@example.com",
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"authenticated"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Add middleware
			app.Use(AuthMiddleware(tt.mockAuth))

			// Add test endpoint
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"status": "authenticated"})
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute request
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("status = %v, want %v", resp.StatusCode, tt.expectedStatus)
			}

			// Check body contains expected string
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("io.ReadAll() error = %v", err)
			}

			if tt.expectedBody != "" {
				bodyStr := string(body)
				if !strings.Contains(bodyStr, tt.expectedBody) {
					t.Errorf("body = %v, want to contain %v", bodyStr, tt.expectedBody)
				}
			}
		})
	}
}

func TestAuthMiddleware_UserContext(t *testing.T) {
	mockAuth := &mockAuthPort{
		validateTokenFunc: func(ctx context.Context, token string) (authcontract.ValidateTokenResponse, error) {
			return authcontract.ValidateTokenResponse{
				Valid:  true,
				UserID: "user-456",
				Email:  "context@example.com",
			}, nil
		},
	}

	app := fiber.New()
	app.Use(AuthMiddleware(mockAuth))

	// Add endpoint that checks user context
	var capturedClaims *Claims
	app.Get("/test", func(c *fiber.Ctx) error {
		claims, ok := c.Locals(UserContextKey).(*Claims)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "no claims"})
		}
		capturedClaims = claims
		return c.JSON(fiber.Map{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	if capturedClaims == nil {
		t.Fatal("claims not set in context")
	}

	if capturedClaims.UserID != "user-456" {
		t.Errorf("claims.UserID = %v, want %v", capturedClaims.UserID, "user-456")
	}

	if capturedClaims.Email != "context@example.com" {
		t.Errorf("claims.Email = %v, want %v", capturedClaims.Email, "context@example.com")
	}
}
