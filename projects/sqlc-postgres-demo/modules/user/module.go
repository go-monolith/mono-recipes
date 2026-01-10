package user

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserModule provides user management services via sqlc + PostgreSQL.
// It follows SOLID principles:
// - SRP: Module handles lifecycle and framework integration, delegates business logic to UserService
// - OCP: New functionality can be added by extending interfaces without modifying existing code
// - LSP: Any implementation of UserService can be substituted
// - ISP: Small, focused interfaces (UserRepository, UserService)
// - DIP: Module depends on abstractions (interfaces), not concrete implementations
type UserModule struct {
	pool    *pgxpool.Pool
	service UserService
	dbURL   string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*UserModule)(nil)
	_ mono.ServiceProviderModule = (*UserModule)(nil)
	_ mono.HealthCheckableModule = (*UserModule)(nil)
)

// NewModule creates a new UserModule with default configuration.
// Database URL is read from DATABASE_URL environment variable.
func NewModule() *UserModule {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://demo:demo123@localhost:5432/users_db?sslmode=disable"
	}
	return &UserModule{
		dbURL: dbURL,
	}
}

// NewModuleWithService creates a new UserModule with a custom service.
// This constructor enables dependency injection for testing.
func NewModuleWithService(service UserService) *UserModule {
	return &UserModule{
		service: service,
	}
}

// Name returns the module name.
func (m *UserModule) Name() string {
	return "user"
}

// Health performs a health check on the user module.
func (m *UserModule) Health(ctx context.Context) mono.HealthStatus {
	if m.pool == nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: "database pool not initialized",
		}
	}

	if err := m.pool.Ping(ctx); err != nil {
		return mono.HealthStatus{
			Healthy: false,
			Message: fmt.Sprintf("database ping failed: %v", err),
		}
	}

	return mono.HealthStatus{
		Healthy: true,
		Message: "operational",
		Details: map[string]any{
			"driver": "pgx/v5",
			"pool":   "postgresql",
		},
	}
}

// RegisterServices registers request-reply services in the service container.
// The framework automatically prefixes service names with "services.<module>."
// so "create" becomes "services.user.create" in the NATS subject.
func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
	if err := helper.RegisterTypedRequestReplyService(
		container, "create", json.Unmarshal, json.Marshal, m.handleCreate,
	); err != nil {
		return fmt.Errorf("failed to register create service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "get", json.Unmarshal, json.Marshal, m.handleGet,
	); err != nil {
		return fmt.Errorf("failed to register get service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "list", json.Unmarshal, json.Marshal, m.handleList,
	); err != nil {
		return fmt.Errorf("failed to register list service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "update", json.Unmarshal, json.Marshal, m.handleUpdate,
	); err != nil {
		return fmt.Errorf("failed to register update service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "delete", json.Unmarshal, json.Marshal, m.handleDelete,
	); err != nil {
		return fmt.Errorf("failed to register delete service: %w", err)
	}

	log.Printf("[user] Registered services: services.user.{create,get,list,update,delete}")
	return nil
}

// Start initializes the database connection pool and creates the service layer.
func (m *UserModule) Start(ctx context.Context) error {
	// Skip database initialization if service is already injected (for testing)
	if m.service != nil {
		log.Println("[user] Module started with injected service")
		return nil
	}

	log.Printf("[user] Connecting to PostgreSQL...")

	pool, err := pgxpool.New(ctx, m.dbURL)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.pool = pool

	// Create repository and service (dependency injection)
	repo := NewPostgresRepository(pool)
	m.service = NewUserService(repo)

	log.Println("[user] Module started successfully")
	return nil
}

// Stop gracefully closes the database connection pool.
func (m *UserModule) Stop(_ context.Context) error {
	if m.pool == nil {
		return nil
	}

	log.Println("[user] Closing database connection pool...")
	m.pool.Close()
	log.Println("[user] Database connection pool closed")
	return nil
}

// Handler methods delegate to the service layer.
// These thin handlers follow SRP by only handling request/response mapping.

func (m *UserModule) handleCreate(ctx context.Context, req CreateUserRequest, _ *mono.Msg) (UserResponse, error) {
	return m.service.Create(ctx, req)
}

func (m *UserModule) handleGet(ctx context.Context, req GetUserRequest, _ *mono.Msg) (UserResponse, error) {
	return m.service.Get(ctx, req)
}

func (m *UserModule) handleList(ctx context.Context, req ListUsersRequest, _ *mono.Msg) (ListUsersResponse, error) {
	return m.service.List(ctx, req)
}

func (m *UserModule) handleUpdate(ctx context.Context, req UpdateUserRequest, _ *mono.Msg) (UserResponse, error) {
	return m.service.Update(ctx, req)
}

func (m *UserModule) handleDelete(ctx context.Context, req DeleteUserRequest, _ *mono.Msg) (DeleteUserResponse, error) {
	return m.service.Delete(ctx, req)
}
