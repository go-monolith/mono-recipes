package article

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ArticleModule provides article management services via sqlc + Prisma Postgres.
type ArticleModule struct {
	pool    *pgxpool.Pool
	service ArticleService
	dbURL   string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*ArticleModule)(nil)
	_ mono.ServiceProviderModule = (*ArticleModule)(nil)
	_ mono.HealthCheckableModule = (*ArticleModule)(nil)
)

// NewModule creates a new ArticleModule with default configuration.
// Database URL is read from DATABASE_URL environment variable.
func NewModule() *ArticleModule {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default Prisma Postgres local development URL
		dbURL = "postgres://postgres:postgres@localhost:51214/template1?sslmode=disable&connection_limit=1&connect_timeout=0&max_idle_connection_lifetime=0&pool_timeout=0&single_use_connections=true&socket_timeout=0"
		log.Println("[article] WARNING: DATABASE_URL not set, using development default")
	}
	return &ArticleModule{
		dbURL: dbURL,
	}
}

// NewModuleWithService creates a new ArticleModule with a custom service.
// This constructor enables dependency injection for testing.
func NewModuleWithService(service ArticleService) *ArticleModule {
	return &ArticleModule{
		service: service,
	}
}

// Name returns the module name.
func (m *ArticleModule) Name() string {
	return "article"
}

// Health performs a health check on the article module.
func (m *ArticleModule) Health(ctx context.Context) mono.HealthStatus {
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
			"driver":   "pgx/v5",
			"database": "prisma-postgres",
		},
	}
}

// RegisterServices registers request-reply services in the service container.
// The framework automatically prefixes service names with "services.<module>."
// so "create" becomes "services.article.create" in the NATS subject.
func (m *ArticleModule) RegisterServices(container mono.ServiceContainer) error {
	if err := helper.RegisterTypedRequestReplyService(
		container, "create", json.Unmarshal, json.Marshal, m.handleCreate,
	); err != nil {
		return fmt.Errorf("register create: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "get", json.Unmarshal, json.Marshal, m.handleGet,
	); err != nil {
		return fmt.Errorf("register get: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "list", json.Unmarshal, json.Marshal, m.handleList,
	); err != nil {
		return fmt.Errorf("register list: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "update", json.Unmarshal, json.Marshal, m.handleUpdate,
	); err != nil {
		return fmt.Errorf("register update: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "delete", json.Unmarshal, json.Marshal, m.handleDelete,
	); err != nil {
		return fmt.Errorf("register delete: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "publish", json.Unmarshal, json.Marshal, m.handlePublish,
	); err != nil {
		return fmt.Errorf("register publish: %w", err)
	}

	log.Printf("[article] Registered services: services.article.{create,get,list,update,delete,publish}")
	return nil
}

// Start initializes the database connection pool and creates the service layer.
func (m *ArticleModule) Start(ctx context.Context) error {
	// Skip database initialization if service is already injected (for testing)
	if m.service != nil {
		log.Println("[article] Module started with injected service")
		return nil
	}

	log.Printf("[article] Connecting to Prisma Postgres...")

	// Parse the database URL to get a config we can modify
	config, err := pgxpool.ParseConfig(m.dbURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Disable prepared statement caching to avoid conflicts with PGlite
	// PGlite (Prisma's local PostgreSQL) has issues with pgx's statement cache
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
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
	m.service = NewArticleService(repo)

	log.Println("[article] Module started successfully")
	return nil
}

// Stop gracefully closes the database connection pool.
func (m *ArticleModule) Stop(_ context.Context) error {
	if m.pool == nil {
		return nil
	}

	log.Println("[article] Closing database connection pool...")
	m.pool.Close()
	log.Println("[article] Database connection pool closed")
	return nil
}

// Handler methods delegate to the service layer.

func (m *ArticleModule) handleCreate(ctx context.Context, req CreateArticleRequest, _ *mono.Msg) (ArticleResponse, error) {
	return m.service.Create(ctx, req)
}

func (m *ArticleModule) handleGet(ctx context.Context, req GetArticleRequest, _ *mono.Msg) (ArticleResponse, error) {
	return m.service.Get(ctx, req)
}

func (m *ArticleModule) handleList(ctx context.Context, req ListArticlesRequest, _ *mono.Msg) (ListArticlesResponse, error) {
	return m.service.List(ctx, req)
}

func (m *ArticleModule) handleUpdate(ctx context.Context, req UpdateArticleRequest, _ *mono.Msg) (ArticleResponse, error) {
	return m.service.Update(ctx, req)
}

func (m *ArticleModule) handleDelete(ctx context.Context, req DeleteArticleRequest, _ *mono.Msg) (DeleteArticleResponse, error) {
	return m.service.Delete(ctx, req)
}

func (m *ArticleModule) handlePublish(ctx context.Context, req PublishArticleRequest, _ *mono.Msg) (ArticleResponse, error) {
	return m.service.Publish(ctx, req)
}
