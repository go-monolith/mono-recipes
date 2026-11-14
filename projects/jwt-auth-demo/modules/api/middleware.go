package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	authcontract "github.com/example/jwt-auth-demo/modules/auth/contract"
)

const (
	// UserContextKey is the key used to store user claims in the Fiber context.
	UserContextKey = "user"
)

// AuthMiddleware creates a middleware that validates JWT tokens through the
// auth module's validate-token contract and stores the caller's *Claims in
// c.Locals(UserContextKey).
func AuthMiddleware(authPort AuthPort) fiber.Handler {
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
		result, err := authPort.ValidateToken(c.UserContext(), authcontract.ValidateTokenRequest{Token: token})
		if err != nil || !result.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid or expired token",
			})
		}

		// Store claims in context for use in handlers
		c.Locals(UserContextKey, &Claims{UserID: result.UserID, Email: result.Email})

		return c.Next()
	}
}
