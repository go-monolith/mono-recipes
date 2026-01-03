package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired.
	ErrExpiredToken = errors.New("token has expired")
)

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// DefaultJWTConfig returns a default JWT configuration.
// In production, the secret key should be loaded from environment variables.
func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		SecretKey:            "your-secret-key-change-in-production",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "jwt-auth-demo",
	}
}

// JWTClaims represents the custom claims for JWT tokens.
type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations.
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWTManager with the given configuration.
func NewJWTManager(config JWTConfig) *JWTManager {
	return &JWTManager{
		config: config,
	}
}

// GenerateAccessToken generates a new access token for the given user.
func (m *JWTManager) GenerateAccessToken(userID, email string) (string, error) {
	return m.generateToken(userID, email, "access", m.config.AccessTokenDuration)
}

// GenerateRefreshToken generates a new refresh token for the given user.
func (m *JWTManager) GenerateRefreshToken(userID, email string) (string, error) {
	return m.generateToken(userID, email, "refresh", m.config.RefreshTokenDuration)
}

// generateToken creates a new JWT token with the specified parameters.
func (m *JWTManager) generateToken(userID, email, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken validates the token and returns the claims if valid.
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAccessToken validates an access token.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token.
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// AccessTokenDuration returns the access token duration in seconds.
func (m *JWTManager) AccessTokenDuration() int64 {
	return int64(m.config.AccessTokenDuration.Seconds())
}
