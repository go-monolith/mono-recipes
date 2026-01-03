package api

import (
	"strings"

	"github.com/example/jwt-auth-demo/modules/auth"
	"github.com/gofiber/fiber/v2"
)

const (
	// UserContextKey is the key used to store user claims in the Fiber context.
	UserContextKey = "user"
)

// AuthMiddleware creates a middleware that validates JWT tokens.
func AuthMiddleware(authAdapter auth.AuthPort) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "unauthorized",
				Message: "Authorization header is required",
			})
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid authorization header format. Use: Bearer <token>",
			})
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "unauthorized",
				Message: "Token is required",
			})
		}

		// Validate token
		claims, err := authAdapter.ValidateToken(c.UserContext(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
		}

		// Store claims in context for use in handlers
		c.Locals(UserContextKey, claims)

		return c.Next()
	}
}
