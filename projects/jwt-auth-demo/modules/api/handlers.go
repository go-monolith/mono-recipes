package api

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

// Handlers contains HTTP handlers for the API.
type Handlers struct {
	auth AuthPort
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(auth AuthPort) *Handlers {
	return &Handlers{auth: auth}
}

// Register handles user registration.
func (h *Handlers) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Email and password are required",
		})
	}

	// Call auth service
	resp, err := h.auth.Register(c.UserContext(), authcontract.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(UserResponse{
		ID:        resp.ID,
		Email:     resp.Email,
		CreatedAt: resp.CreatedAt,
	})
}

// Login handles user login.
func (h *Handlers) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Email and password are required",
		})
	}

	// Call auth service
	resp, err := h.auth.Login(c.UserContext(), authcontract.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(TokenResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		TokenType:    resp.TokenType,
	})
}

// Refresh handles token refresh.
func (h *Handlers) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Refresh token is required",
		})
	}

	// Call auth service
	resp, err := h.auth.RefreshToken(c.UserContext(), authcontract.RefreshRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid or expired refresh token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(TokenResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		TokenType:    resp.TokenType,
	})
}

// Profile handles getting the current user's profile.
// This is a protected endpoint that requires a valid JWT token.
func (h *Handlers) Profile(c *fiber.Ctx) error {
	// Get user claims from context (set by auth middleware)
	claims, ok := c.Locals(UserContextKey).(*Claims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
	}

	// Get full user details from auth service
	user, err := h.auth.GetUser(c.UserContext(), authcontract.GetUserRequest{UserID: claims.UserID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve user profile",
		})
	}

	return c.Status(fiber.StatusOK).JSON(ProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		Message:   "Welcome! You have accessed a protected resource.",
	})
}

// handleAuthError handles authentication errors and returns appropriate responses.
// It matches error messages to provide user-friendly responses without exposing internals.
func (h *Handlers) handleAuthError(c *fiber.Ctx, err error) error {
	errStr := err.Error()

	// Check for specific error types by matching known error messages
	switch {
	case strings.Contains(errStr, "invalid email or password"):
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid email or password",
		})
	case strings.Contains(errStr, "user with this email already exists"):
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
			Error:   "conflict",
			Message: "User with this email already exists",
		})
	case strings.Contains(errStr, "invalid email format"):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid email format",
		})
	case strings.Contains(errStr, "password must be at least"):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Password must be at least 8 characters",
		})
	case strings.Contains(errStr, "password must be at most"):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "bad_request",
			Message: "Password must be at most 72 characters",
		})
	default:
		// Log the actual error but don't expose it to the client
		log.Printf("[api] Internal error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "internal_error",
			Message: "An internal error occurred",
		})
	}
}
