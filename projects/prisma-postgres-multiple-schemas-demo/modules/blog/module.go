package blog

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

// BlogModule provides blog management services via sqlc + Prisma Postgres.
type BlogModule struct {
	pool           *pgxpool.Pool
	postService    PostService
	commentService CommentService
	dbURL          string
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*BlogModule)(nil)
	_ mono.ServiceProviderModule = (*BlogModule)(nil)
	_ mono.HealthCheckableModule = (*BlogModule)(nil)
)

// NewModule creates a new BlogModule with default configuration.
// Database URL is read from DATABASE_URL environment variable.
func NewModule() *BlogModule {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:51214/template1?schema=blog&sslmode=disable&connection_limit=1&connect_timeout=0&max_idle_connection_lifetime=0&pool_timeout=0&single_use_connections=true&socket_timeout=0"
		log.Println("[blog] WARNING: DATABASE_URL not set, using development default")
	}
	return &BlogModule{
		dbURL: dbURL,
	}
}

// NewModuleWithServices creates a new BlogModule with custom services.
// This constructor enables dependency injection for testing.
func NewModuleWithServices(postService PostService, commentService CommentService) *BlogModule {
	return &BlogModule{
		postService:    postService,
		commentService: commentService,
	}
}

// Name returns the module name.
func (m *BlogModule) Name() string {
	return "blog"
}

// Health performs a health check on the blog module.
func (m *BlogModule) Health(ctx context.Context) mono.HealthStatus {
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
// so "post.create" becomes "services.blog.post.create" in the NATS subject.
func (m *BlogModule) RegisterServices(container mono.ServiceContainer) error {
	// Post services
	if err := helper.RegisterTypedRequestReplyService(
		container, "post.create", json.Unmarshal, json.Marshal, m.handlePostCreate,
	); err != nil {
		return fmt.Errorf("register post.create: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "post.get", json.Unmarshal, json.Marshal, m.handlePostGet,
	); err != nil {
		return fmt.Errorf("register post.get: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "post.list", json.Unmarshal, json.Marshal, m.handlePostList,
	); err != nil {
		return fmt.Errorf("register post.list: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "post.update", json.Unmarshal, json.Marshal, m.handlePostUpdate,
	); err != nil {
		return fmt.Errorf("register post.update: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "post.delete", json.Unmarshal, json.Marshal, m.handlePostDelete,
	); err != nil {
		return fmt.Errorf("register post.delete: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "post.publish", json.Unmarshal, json.Marshal, m.handlePostPublish,
	); err != nil {
		return fmt.Errorf("register post.publish: %w", err)
	}

	// Comment services
	if err := helper.RegisterTypedRequestReplyService(
		container, "comment.create", json.Unmarshal, json.Marshal, m.handleCommentCreate,
	); err != nil {
		return fmt.Errorf("register comment.create: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "comment.get", json.Unmarshal, json.Marshal, m.handleCommentGet,
	); err != nil {
		return fmt.Errorf("register comment.get: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "comment.list", json.Unmarshal, json.Marshal, m.handleCommentList,
	); err != nil {
		return fmt.Errorf("register comment.list: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "comment.update", json.Unmarshal, json.Marshal, m.handleCommentUpdate,
	); err != nil {
		return fmt.Errorf("register comment.update: %w", err)
	}
	if err := helper.RegisterTypedRequestReplyService(
		container, "comment.delete", json.Unmarshal, json.Marshal, m.handleCommentDelete,
	); err != nil {
		return fmt.Errorf("register comment.delete: %w", err)
	}

	log.Printf("[blog] Registered services: services.blog.{post,comment}.{create,get,list,update,delete} + post.publish")
	return nil
}

// Start initializes the database connection pool and creates the service layer.
func (m *BlogModule) Start(ctx context.Context) error {
	// Skip database initialization if services are already injected (for testing)
	if m.postService != nil && m.commentService != nil {
		log.Println("[blog] Module started with injected services")
		return nil
	}

	log.Printf("[blog] Connecting to Prisma Postgres...")

	config, err := pgxpool.ParseConfig(m.dbURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Disable prepared statement caching to avoid conflicts with PGlite
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.pool = pool

	// Create repositories and services (dependency injection)
	postRepo := NewPostgresPostRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	m.postService = NewPostService(postRepo)
	m.commentService = NewCommentService(commentRepo, postRepo)

	log.Println("[blog] Module started successfully")
	return nil
}

// Stop gracefully closes the database connection pool.
func (m *BlogModule) Stop(_ context.Context) error {
	if m.pool == nil {
		return nil
	}

	log.Println("[blog] Closing database connection pool...")
	m.pool.Close()
	log.Println("[blog] Database connection pool closed")
	return nil
}

// Post handler methods

func (m *BlogModule) handlePostCreate(ctx context.Context, req CreatePostRequest, _ *mono.Msg) (PostResponse, error) {
	return m.postService.Create(ctx, req)
}

func (m *BlogModule) handlePostGet(ctx context.Context, req GetPostRequest, _ *mono.Msg) (PostResponse, error) {
	return m.postService.Get(ctx, req)
}

func (m *BlogModule) handlePostList(ctx context.Context, req ListPostsRequest, _ *mono.Msg) (ListPostsResponse, error) {
	return m.postService.List(ctx, req)
}

func (m *BlogModule) handlePostUpdate(ctx context.Context, req UpdatePostRequest, _ *mono.Msg) (PostResponse, error) {
	return m.postService.Update(ctx, req)
}

func (m *BlogModule) handlePostDelete(ctx context.Context, req DeletePostRequest, _ *mono.Msg) (DeletePostResponse, error) {
	return m.postService.Delete(ctx, req)
}

func (m *BlogModule) handlePostPublish(ctx context.Context, req PublishPostRequest, _ *mono.Msg) (PostResponse, error) {
	return m.postService.Publish(ctx, req)
}

// Comment handler methods

func (m *BlogModule) handleCommentCreate(ctx context.Context, req CreateCommentRequest, _ *mono.Msg) (CommentResponse, error) {
	return m.commentService.Create(ctx, req)
}

func (m *BlogModule) handleCommentGet(ctx context.Context, req GetCommentRequest, _ *mono.Msg) (CommentResponse, error) {
	return m.commentService.Get(ctx, req)
}

func (m *BlogModule) handleCommentList(ctx context.Context, req ListCommentsRequest, _ *mono.Msg) (ListCommentsResponse, error) {
	return m.commentService.List(ctx, req)
}

func (m *BlogModule) handleCommentUpdate(ctx context.Context, req UpdateCommentRequest, _ *mono.Msg) (CommentResponse, error) {
	return m.commentService.Update(ctx, req)
}

func (m *BlogModule) handleCommentDelete(ctx context.Context, req DeleteCommentRequest, _ *mono.Msg) (DeleteCommentResponse, error) {
	return m.commentService.Delete(ctx, req)
}
