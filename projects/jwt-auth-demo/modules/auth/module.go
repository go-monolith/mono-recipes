package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	domain "github.com/example/jwt-auth-demo/domain/user"
	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// AuthModule provides authentication services.
type AuthModule struct {
	db      *gorm.DB
	service *AuthService
	dbPath  string
}

// Compile-time interface checks.
var _ mono.Module = (*AuthModule)(nil)
var _ mono.ServiceProviderModule = (*AuthModule)(nil)
var _ mono.HealthCheckableModule = (*AuthModule)(nil)

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
		sqlDB, err := m.db.DB()
		if err == nil {
			sqlDB.Close()
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

// RegisterServices registers request-reply services in the service container.
func (m *AuthModule) RegisterServices(container mono.ServiceContainer) error {
	// Register register service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"register",
		json.Unmarshal,
		json.Marshal,
		m.handleRegister,
	); err != nil {
		return fmt.Errorf("failed to register register service: %w", err)
	}

	// Register login service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"login",
		json.Unmarshal,
		json.Marshal,
		m.handleLogin,
	); err != nil {
		return fmt.Errorf("failed to register login service: %w", err)
	}

	// Register refresh-token service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"refresh-token",
		json.Unmarshal,
		json.Marshal,
		m.handleRefresh,
	); err != nil {
		return fmt.Errorf("failed to register refresh-token service: %w", err)
	}

	// Register validate-token service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"validate-token",
		json.Unmarshal,
		json.Marshal,
		m.handleValidateToken,
	); err != nil {
		return fmt.Errorf("failed to register validate-token service: %w", err)
	}

	// Register get-user service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"get-user",
		json.Unmarshal,
		json.Marshal,
		m.handleGetUser,
	); err != nil {
		return fmt.Errorf("failed to register get-user service: %w", err)
	}

	log.Printf("[auth] Registered services: register, login, refresh-token, validate-token, get-user")
	return nil
}

// handleRegister handles user registration.
func (m *AuthModule) handleRegister(ctx context.Context, req RegisterRequest, _ *mono.Msg) (RegisterResponse, error) {
	user, err := m.service.Register(ctx, req.Email, req.Password)
	if err != nil {
		return RegisterResponse{}, err
	}

	return RegisterResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// handleLogin handles user login.
func (m *AuthModule) handleLogin(ctx context.Context, req LoginRequest, _ *mono.Msg) (LoginResponse, error) {
	tokens, err := m.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
	}, nil
}

// handleRefresh handles token refresh.
func (m *AuthModule) handleRefresh(ctx context.Context, req RefreshRequest, _ *mono.Msg) (RefreshResponse, error) {
	tokens, err := m.service.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return RefreshResponse{}, err
	}

	return RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
	}, nil
}

// handleValidateToken handles token validation.
func (m *AuthModule) handleValidateToken(ctx context.Context, req ValidateTokenRequest, _ *mono.Msg) (ValidateTokenResponse, error) {
	claims, err := m.service.ValidateToken(ctx, req.Token)
	if err != nil {
		errMsg := "invalid token"
		if errors.Is(err, ErrExpiredToken) {
			errMsg = "token expired"
		}
		return ValidateTokenResponse{
			Valid: false,
			Error: errMsg,
		}, nil // Return response, not error, for validation failures
	}

	return ValidateTokenResponse{
		Valid:  true,
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

// handleGetUser handles get user requests.
func (m *AuthModule) handleGetUser(ctx context.Context, req GetUserRequest, _ *mono.Msg) (GetUserResponse, error) {
	user, err := m.service.GetUser(ctx, req.UserID)
	if err != nil {
		return GetUserResponse{}, err
	}

	return GetUserResponse{
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
