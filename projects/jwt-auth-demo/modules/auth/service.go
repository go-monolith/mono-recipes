package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	domain "github.com/example/jwt-auth-demo/domain/user"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials is returned when login credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrInvalidEmail is returned when email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")
	// ErrWeakPassword is returned when password is too weak.
	ErrWeakPassword = errors.New("password must be at least 8 characters")
	// ErrPasswordTooLong is returned when password exceeds bcrypt's 72-byte limit.
	ErrPasswordTooLong = errors.New("password must be at most 72 characters")
)

// AuthService handles authentication business logic.
type AuthService struct {
	repo     *UserRepository
	hasher   *PasswordHasher
	jwt      *JWTManager
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *UserRepository, hasher *PasswordHasher, jwt *JWTManager) *AuthService {
	return &AuthService{
		repo:   repo,
		hasher: hasher,
		jwt:    jwt,
	}
}

// Register creates a new user account.
func (s *AuthService) Register(_ context.Context, email, password string) (*domain.User, error) {
	// Validate email using standard library
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}

	// Validate password length (bcrypt has 72-byte limit)
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}
	if len(password) > 72 {
		return nil, ErrPasswordTooLong
	}

	// Check if user already exists
	exists, err := s.repo.EmailExists(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	// Hash password
	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(_ context.Context, email, password string) (*domain.TokenPair, error) {
	// Find user by email
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if !s.hasher.Verify(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	return s.generateTokenPair(user.ID, user.Email)
}

// RefreshTokens generates new access and refresh tokens.
func (s *AuthService) RefreshTokens(_ context.Context, refreshToken string) (*domain.TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify user still exists
	user, err := s.repo.FindByID(claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate new tokens
	return s.generateTokenPair(user.ID, user.Email)
}

// ValidateToken validates an access token and returns claims.
func (s *AuthService) ValidateToken(_ context.Context, token string) (*domain.Claims, error) {
	claims, err := s.jwt.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}

	return &domain.Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

// GetUser retrieves a user by ID.
func (s *AuthService) GetUser(_ context.Context, userID string) (*domain.User, error) {
	return s.repo.FindByID(userID)
}

// generateTokenPair generates both access and refresh tokens.
func (s *AuthService) generateTokenPair(userID, email string) (*domain.TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccessToken(userID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(userID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwt.AccessTokenDuration(),
		TokenType:    "Bearer",
	}, nil
}

