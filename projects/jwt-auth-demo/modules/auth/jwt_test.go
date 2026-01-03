package auth

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateAndValidateAccessToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:            "test-secret-key",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	manager := NewJWTManager(config)

	userID := "user-123"
	email := "test@example.com"

	// Generate access token
	token, err := manager.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	// Validate access token
	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("claims.Email = %v, want %v", claims.Email, email)
	}
	if claims.TokenType != "access" {
		t.Errorf("claims.TokenType = %v, want %v", claims.TokenType, "access")
	}
	if claims.Issuer != config.Issuer {
		t.Errorf("claims.Issuer = %v, want %v", claims.Issuer, config.Issuer)
	}
}

func TestJWTManager_GenerateAndValidateRefreshToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:            "test-secret-key",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	manager := NewJWTManager(config)

	userID := "user-456"
	email := "refresh@example.com"

	// Generate refresh token
	token, err := manager.GenerateRefreshToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	// Validate refresh token
	claims, err := manager.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.TokenType != "refresh" {
		t.Errorf("claims.TokenType = %v, want %v", claims.TokenType, "refresh")
	}
}

func TestJWTManager_AccessTokenCannotBeUsedAsRefresh(t *testing.T) {
	config := DefaultJWTConfig()
	manager := NewJWTManager(config)

	// Generate access token
	accessToken, err := manager.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Try to validate as refresh token
	_, err = manager.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Error("ValidateRefreshToken() should reject access token")
	}
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManager_RefreshTokenCannotBeUsedAsAccess(t *testing.T) {
	config := DefaultJWTConfig()
	manager := NewJWTManager(config)

	// Generate refresh token
	refreshToken, err := manager.GenerateRefreshToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	// Try to validate as access token
	_, err = manager.ValidateAccessToken(refreshToken)
	if err == nil {
		t.Error("ValidateAccessToken() should reject refresh token")
	}
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManager_InvalidToken(t *testing.T) {
	config := DefaultJWTConfig()
	manager := NewJWTManager(config)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "random string",
			token: "not.a.valid.token",
		},
		{
			name:  "malformed jwt",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ValidateToken(tt.token)
			if err == nil {
				t.Error("ValidateToken() should return error for invalid token")
			}
		})
	}
}

func TestJWTManager_WrongSecretKey(t *testing.T) {
	config1 := JWTConfig{
		SecretKey:            "secret-key-1",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	config2 := JWTConfig{
		SecretKey:            "secret-key-2",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-issuer",
	}

	manager1 := NewJWTManager(config1)
	manager2 := NewJWTManager(config2)

	// Generate token with manager1
	token, err := manager1.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Try to validate with manager2 (different secret)
	_, err = manager2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail with different secret key")
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:            "test-secret-key",
		AccessTokenDuration:  1 * time.Millisecond, // Very short duration
		RefreshTokenDuration: 1 * time.Millisecond,
		Issuer:               "test-issuer",
	}
	manager := NewJWTManager(config)

	token, err := manager.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail for expired token")
	}
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestJWTManager_AccessTokenDuration(t *testing.T) {
	config := JWTConfig{
		SecretKey:            "test-secret-key",
		AccessTokenDuration:  30 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	manager := NewJWTManager(config)

	expected := int64(30 * 60) // 30 minutes in seconds
	got := manager.AccessTokenDuration()

	if got != expected {
		t.Errorf("AccessTokenDuration() = %v, want %v", got, expected)
	}
}
