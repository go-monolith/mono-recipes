// Package auth holds the auth module: user registration, login and JWT
// handling backed by SQLite (GORM). Other modules call it through
// auth/contract, never by importing this package.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/go-monolith/mono"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	domain "github.com/example/jwt-auth-demo/domain/user"
	"github.com/example/jwt-auth-demo/modules/auth/contract"
)

// AuthModule provides authentication services.
type AuthModule struct {
	db      *gorm.DB
	service *AuthService
	dbPath  string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*AuthModule)(nil)
	_ mono.ServiceProviderModule = (*AuthModule)(nil)
	_ mono.HealthCheckableModule = (*AuthModule)(nil)
)

// NewModule creates a new AuthModule.
func NewModule() *AuthModule {
	// Use environment variable for DB path, default to local file
	dbPath := os.Getenv("JWT_AUTH_DB_PATH")
	if dbPath == "" {
		dbPath = "jwt_auth.db"
	}
	return &AuthModule{
		dbPath: dbPath,
	}
}

// Name returns the module name.
func (m *AuthModule) Name() string {
	return "auth"
}

// Start initializes the auth module.
func (m *AuthModule) Start(_ context.Context) error {
	// Initialize SQLite database with GORM
	db, err := gorm.Open(sqlite.Open(m.dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	m.db = db

	// Auto-migrate the User schema
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Initialize components
	repo := NewUserRepository(db)
	hasher := NewPasswordHasher()

	// Load JWT config from environment or use defaults
	jwtConfig := loadJWTConfig()
	jwtManager := NewJWTManager(jwtConfig)

	m.service = NewAuthService(repo, hasher, jwtManager)

	log.Printf("[auth] Module started (database: %s)", m.dbPath)
	return nil
}

// Stop shuts down the module.
func (m *AuthModule) Stop(_ context.Context) error {
	if m.db != nil {
		if sqlDB, err := m.db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	log.Println("[auth] Module stopped")
	return nil
}

// Health returns the health status of the module.
func (m *AuthModule) Health(_ context.Context) mono.HealthStatus {
	if m.db == nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: "database not initialized",
		}
	}

	sqlDB, err := m.db.DB()
	if err != nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("failed to get database connection: %v", err),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("database ping failed: %v", err),
		}
	}

	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"database": m.dbPath,
		},
	}
}

// RegisterServices is the driving adapter: it binds the handlers below to the
// module's own contracts. No service-name strings, no manual JSON.
func (m *AuthModule) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.Register.Register(container, m.handleRegister); err != nil {
		return fmt.Errorf("failed to register %s service: %w", contract.Register.Name, err)
	}
	if err := contract.Login.Register(container, m.handleLogin); err != nil {
		return fmt.Errorf("failed to register %s service: %w", contract.Login.Name, err)
	}
	if err := contract.RefreshToken.Register(container, m.handleRefresh); err != nil {
		return fmt.Errorf("failed to register %s service: %w", contract.RefreshToken.Name, err)
	}
	if err := contract.ValidateToken.Register(container, m.handleValidateToken); err != nil {
		return fmt.Errorf("failed to register %s service: %w", contract.ValidateToken.Name, err)
	}
	if err := contract.GetUser.Register(container, m.handleGetUser); err != nil {
		return fmt.Errorf("failed to register %s service: %w", contract.GetUser.Name, err)
	}

	log.Printf("[auth] Registered services: %s, %s, %s, %s, %s",
		contract.Register.Name, contract.Login.Name, contract.RefreshToken.Name,
		contract.ValidateToken.Name, contract.GetUser.Name)
	return nil
}

// handleRegister handles user registration.
func (m *AuthModule) handleRegister(ctx context.Context, req contract.RegisterRequest, _ *mono.Msg) (contract.RegisterResponse, error) {
	user, err := m.service.Register(ctx, req.Email, req.Password)
	if err != nil {
		return contract.RegisterResponse{}, err
	}

	return contract.RegisterResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// handleLogin handles user login.
func (m *AuthModule) handleLogin(ctx context.Context, req contract.LoginRequest, _ *mono.Msg) (contract.TokenResponse, error) {
	tokens, err := m.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		return contract.TokenResponse{}, err
	}

	return contract.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
	}, nil
}

// handleRefresh handles token refresh.
func (m *AuthModule) handleRefresh(ctx context.Context, req contract.RefreshRequest, _ *mono.Msg) (contract.TokenResponse, error) {
	tokens, err := m.service.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return contract.TokenResponse{}, err
	}

	return contract.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
	}, nil
}

// handleValidateToken handles token validation.
func (m *AuthModule) handleValidateToken(ctx context.Context, req contract.ValidateTokenRequest, _ *mono.Msg) (contract.ValidateTokenResponse, error) {
	claims, err := m.service.ValidateToken(ctx, req.Token)
	if err != nil {
		errMsg := "invalid token"
		if errors.Is(err, ErrExpiredToken) {
			errMsg = "token expired"
		}
		return contract.ValidateTokenResponse{
			Valid: false,
			Error: errMsg,
		}, nil // Return response, not error, for validation failures
	}

	return contract.ValidateTokenResponse{
		Valid:  true,
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

// handleGetUser handles get user requests.
func (m *AuthModule) handleGetUser(ctx context.Context, req contract.GetUserRequest, _ *mono.Msg) (contract.GetUserResponse, error) {
	user, err := m.service.GetUser(ctx, req.UserID)
	if err != nil {
		return contract.GetUserResponse{}, err
	}

	return contract.GetUserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// loadJWTConfig loads JWT configuration from environment variables.
func loadJWTConfig() JWTConfig {
	config := DefaultJWTConfig()

	if secret := os.Getenv("JWT_SECRET_KEY"); secret != "" {
		config.SecretKey = secret
	}

	if issuer := os.Getenv("JWT_ISSUER"); issuer != "" {
		config.Issuer = issuer
	}

	return config
}
