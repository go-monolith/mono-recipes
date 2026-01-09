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
type UserModule struct {
	pool  *pgxpool.Pool
	repo  *Repository
	dbURL string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*UserModule)(nil)
	_ mono.ServiceProviderModule = (*UserModule)(nil)
	_ mono.HealthCheckableModule = (*UserModule)(nil)
)

// NewModule creates a new UserModule.
func NewModule() *UserModule {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://demo:demo123@localhost:5432/users_db?sslmode=disable"
	}
	return &UserModule{
		dbURL: dbURL,
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
		container, "create", json.Unmarshal, json.Marshal, m.createUser,
	); err != nil {
		return fmt.Errorf("failed to register create service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "get", json.Unmarshal, json.Marshal, m.getUser,
	); err != nil {
		return fmt.Errorf("failed to register get service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "list", json.Unmarshal, json.Marshal, m.listUsers,
	); err != nil {
		return fmt.Errorf("failed to register list service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "update", json.Unmarshal, json.Marshal, m.updateUser,
	); err != nil {
		return fmt.Errorf("failed to register update service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "delete", json.Unmarshal, json.Marshal, m.deleteUser,
	); err != nil {
		return fmt.Errorf("failed to register delete service: %w", err)
	}

	log.Printf("[user] Registered services: services.user.{create,get,list,update,delete}")
	return nil
}

// Start initializes the database connection pool.
func (m *UserModule) Start(ctx context.Context) error {
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
	m.repo = NewRepository(pool)

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
