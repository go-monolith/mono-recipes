# JWT Authentication Demo

A complete JWT authentication implementation using the mono framework, Echo, GORM, and SQLite. This recipe demonstrates secure user authentication with access and refresh tokens.

## What This Recipe Demonstrates

- **Secure Password Hashing**: Using bcrypt with configurable cost factor
- **JWT Token Generation**: Access tokens (15 min) and refresh tokens (7 days)
- **Authentication Middleware**: Protecting routes with Bearer token validation
- **ServiceProviderModule Pattern**: Clean cross-module communication via mono framework
- **SQLite with GORM**: Simple embedded database for user storage

## Why JWT for Stateless Authentication?

JWT (JSON Web Tokens) provides stateless authentication, meaning the server doesn't need to store session data. Each request carries its own authentication proof via the token.

### Benefits

1. **Scalability**: No session storage needed on the server; any server instance can validate tokens
2. **Performance**: No database lookup required for each request (token is self-contained)
3. **Microservices-friendly**: Tokens can be validated by any service with the secret key
4. **Mobile/SPA Support**: Easy to store and include in API requests

### Security Considerations

1. **Token Storage**: Store access tokens in memory (not localStorage) for web apps to prevent XSS attacks
2. **Refresh Token Strategy**: Use HTTP-only cookies for refresh tokens in production
3. **Token Expiry**: Short-lived access tokens (15 min) limit damage from stolen tokens
4. **Secret Key Management**: Use strong, randomly generated secrets; rotate periodically
5. **HTTPS Only**: Always use HTTPS to prevent token interception

### When to Use JWT vs Session-Based Auth

| Use Case | Recommendation |
|----------|----------------|
| Single-page applications (SPA) | JWT |
| Mobile applications | JWT |
| Microservices architecture | JWT |
| Traditional server-rendered web apps | Session-based |
| Applications requiring immediate session invalidation | Session-based |
| Simple monolithic applications | Either works |

## Project Structure

```
jwt-auth-demo/
├── main.go                    # Application entry point
├── domain/
│   └── user/
│       └── entity.go          # User entity and types
├── modules/
│   ├── auth/
│   │   ├── module.go          # Auth ServiceProviderModule
│   │   ├── service.go         # Business logic
│   │   ├── repository.go      # Database access
│   │   ├── jwt.go             # JWT token handling
│   │   ├── password.go        # Password hashing
│   │   ├── adapter.go         # Cross-module adapter
│   │   └── types.go           # Request/response types
│   └── api/
│       ├── module.go          # API HTTP module
│       ├── handlers.go        # HTTP handlers
│       ├── middleware.go      # Auth middleware
│       └── types.go           # API types
├── demo.sh                    # Demo script
└── README.md
```

## Running the Demo

### Prerequisites

- Go 1.25 or later
- curl and jq (for demo script)

### Build and Run

```bash
# Build the application
go build -o bin/jwt-auth-demo .

# Run the server
./bin/jwt-auth-demo
```

### Using the Demo Script

```bash
# Make the demo script executable
chmod +x demo.sh

# Run the full demo (recommended for first-time users)
./demo.sh demo

# Individual commands
./demo.sh register       # Register a new user
./demo.sh login          # Login and get tokens
./demo.sh profile        # Access protected profile endpoint
./demo.sh refresh        # Refresh tokens
./demo.sh no-token       # Try accessing protected route without token
./demo.sh invalid-token  # Try accessing protected route with invalid token
```

## API Endpoints

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Register a new user |
| POST | `/api/v1/auth/login` | Login and get tokens |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| GET | `/health` | Health check |

### Protected Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/profile` | Get current user profile |

## Request/Response Examples

### Register

```bash
curl -X POST http://localhost:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

Response:
```json
{
  "id": "uuid-here",
  "email": "user@example.com",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Login

```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

### Access Protected Route

```bash
curl http://localhost:3000/api/v1/profile \
  -H "Authorization: Bearer <access_token>"
```

Response:
```json
{
  "id": "uuid-here",
  "email": "user@example.com",
  "created_at": "2024-01-01T00:00:00Z",
  "message": "Welcome! You have accessed a protected resource."
}
```

### Refresh Token

```bash
curl -X POST http://localhost:3000/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

## Configuration

Environment variables for customization:

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET_KEY` | (internal default) | Secret key for signing tokens |
| `JWT_ISSUER` | `jwt-auth-demo` | Token issuer claim |
| `JWT_AUTH_DB_PATH` | `jwt_auth.db` | SQLite database file path |

**Important**: In production, always set `JWT_SECRET_KEY` to a strong, randomly generated value.

## Implementation Notes

### Password Hashing

Uses bcrypt with cost factor 12 (configurable). Higher cost = more secure but slower:

```go
hasher := auth.NewPasswordHasher()
hash, err := hasher.Hash("password123")
valid := hasher.Verify("password123", hash)
```

### Token Generation

Tokens include standard claims plus custom user data:

```go
type JWTClaims struct {
    UserID    string `json:"user_id"`
    Email     string `json:"email"`
    TokenType string `json:"token_type"`
    jwt.RegisteredClaims
}
```

### Middleware Integration

The auth middleware extracts and validates the Bearer token, then stores claims in the Fiber context:

```go
claims, ok := c.Locals("user").(*domain.Claims)
if ok {
    // User is authenticated
    userID := claims.UserID
}
```

## Success Criteria

- [x] Full auth flow works via demo.sh
- [x] Invalid tokens are properly rejected
- [x] Protected routes return 401 without valid token
- [x] Refresh tokens can be used to get new access tokens

---

*This recipe is part of the Mono Cookbook collection.*
