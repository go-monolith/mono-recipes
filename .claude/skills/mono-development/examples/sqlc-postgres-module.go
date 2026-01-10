// sqlc-postgres-module.go demonstrates sqlc with PostgreSQL integration
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

// UserModule provides user management via sqlc + PostgreSQL
type UserModule struct {
	pool  *pgxpool.Pool
	dbURL string
}

// Compile-time interface checks
var (
	_ mono.Module                = (*UserModule)(nil)
	_ mono.ServiceProviderModule = (*UserModule)(nil)
	_ mono.HealthCheckableModule = (*UserModule)(nil)
)

// NewModule creates a new UserModule
func NewModule() *UserModule {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://demo:demo123@localhost:5432/users_db?sslmode=disable"
	}
	return &UserModule{
		dbURL: dbURL,
	}
}

// Name returns the module name
func (m *UserModule) Name() string { return "user" }

// Health performs a health check
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

// RegisterServices registers CRUD services
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

// Start initializes the database connection pool
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

	log.Println("[user] Module started successfully")
	return nil
}

// Stop closes the database connection pool
func (m *UserModule) Stop(_ context.Context) error {
	if m.pool == nil {
		return nil
	}

	log.Println("[user] Closing database connection pool...")
	m.pool.Close()
	log.Println("[user] Database connection pool closed")
	return nil
}

// Request/Response types
type CreateUserRequest struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type GetUserRequest struct {
	ID int32 `json:"id"`
}

type ListUsersRequest struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type UpdateUserRequest struct {
	ID     int32  `json:"id"`
	Name   string `json:"name,omitempty"`
	Active *bool  `json:"active,omitempty"`
}

type DeleteUserRequest struct {
	ID int32 `json:"id"`
}

type User struct {
	ID        int32  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UserResponse struct {
	User  *User  `json:"user,omitempty"`
	Error string `json:"error,omitempty"`
}

type UserListResponse struct {
	Users []User `json:"users"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Service handlers using raw SQL (in practice, use sqlc-generated queries)
func (m *UserModule) createUser(ctx context.Context, req CreateUserRequest, _ *mono.Msg) (UserResponse, error) {
	var user User

	err := m.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, active)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, name, active, created_at, updated_at`,
		req.Email, req.Name, req.Active,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return UserResponse{Error: err.Error()}, nil
	}

	return UserResponse{User: &user}, nil
}

func (m *UserModule) getUser(ctx context.Context, req GetUserRequest, _ *mono.Msg) (UserResponse, error) {
	var user User

	err := m.pool.QueryRow(ctx,
		`SELECT id, email, name, active, created_at, updated_at
		 FROM users WHERE id = $1`,
		req.ID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return UserResponse{Error: "user not found"}, nil
	}

	return UserResponse{User: &user}, nil
}

func (m *UserModule) listUsers(ctx context.Context, req ListUsersRequest, _ *mono.Msg) (UserListResponse, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := m.pool.Query(ctx,
		`SELECT id, email, name, active, created_at, updated_at
		 FROM users ORDER BY id LIMIT $1 OFFSET $2`,
		limit, req.Offset,
	)
	if err != nil {
		return UserListResponse{Users: []User{}}, nil
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt); err != nil {
			continue
		}
		users = append(users, user)
	}

	return UserListResponse{Users: users}, nil
}

func (m *UserModule) updateUser(ctx context.Context, req UpdateUserRequest, _ *mono.Msg) (UserResponse, error) {
	var user User

	err := m.pool.QueryRow(ctx,
		`UPDATE users
		 SET name = COALESCE(NULLIF($2, ''), name),
		     active = COALESCE($3, active),
		     updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, email, name, active, created_at, updated_at`,
		req.ID, req.Name, req.Active,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return UserResponse{Error: "user not found"}, nil
	}

	return UserResponse{User: &user}, nil
}

func (m *UserModule) deleteUser(ctx context.Context, req DeleteUserRequest, _ *mono.Msg) (DeleteResponse, error) {
	_, err := m.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, req.ID)
	if err != nil {
		return DeleteResponse{Success: false, Error: err.Error()}, nil
	}

	return DeleteResponse{Success: true}, nil
}

/*
Module-owned database structure (SOLID principle - each module owns its database):

modules/user/
├── module.go
├── service.go
├── repository.go
└── db/                    # Module-owned database
    ├── sqlc.yaml
    ├── schema.sql
    ├── queries/
    │   └── users.sql
    └── generated/
        ├── db.go
        ├── models.go
        └── query.sql.go

modules/user/db/sqlc.yaml:

version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema.sql"
    gen:
      go:
        package: "generated"
        out: "generated"
        sql_package: "pgx/v5"
        emit_json_tags: true

Run: cd modules/user/db && sqlc generate

modules/user/db/schema.sql:

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

modules/user/db/queries/users.sql:

-- name: CreateUser :one
INSERT INTO users (email, name, active)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY id LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET name = $2, active = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
*/
