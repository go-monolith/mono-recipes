package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/prisma-postgres-demo/modules/article"
	"github.com/example/prisma-postgres-demo/modules/blog"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== Prisma + sqlc PostgreSQL Demo ===")
	log.Println("Demonstrating Prisma migrations with sqlc type-safe queries")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register modules
	app.Register(article.NewModule())
	app.Register(blog.NewModule())

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	printStartupInfo()

	// Graceful shutdown
	wait := gfshutdown.GracefulShutdown(
		context.Background(),
		shutdownTimeout,
		map[string]gfshutdown.Operation{
			"mono-app": func(ctx context.Context) error {
				log.Println("Graceful shutdown initiated...")
				return app.Stop(ctx)
			},
		},
	)

	exitCode := <-wait
	log.Printf("Application exited with code: %d", exitCode)
	os.Exit(exitCode)
}

func printStartupInfo() {
	log.Print(`
Application started successfully!

This demo shows:
  - Prisma for local PostgreSQL development (PGlite)
  - Prisma migrations for schema management
  - sqlc type-safe SQL code generation
  - ServiceProviderModule pattern for request-reply services
  - Multiple PostgreSQL schemas (article_module, blog)
  - No HTTP endpoints - pure service-based architecture

Available Services (via NATS request-reply):

  Article Module:
  - services.article.create  - Create a new article
  - services.article.get     - Get article by ID or slug
  - services.article.list    - List articles with pagination
  - services.article.update  - Update article by ID
  - services.article.delete  - Delete article by ID
  - services.article.publish - Publish a draft article

  Blog Module (Posts):
  - services.blog.post.create  - Create a new post
  - services.blog.post.get     - Get post by ID or slug
  - services.blog.post.list    - List posts with pagination
  - services.blog.post.update  - Update post by ID
  - services.blog.post.delete  - Delete post by ID
  - services.blog.post.publish - Publish a draft post

  Blog Module (Comments):
  - services.blog.comment.create - Create a comment on a post
  - services.blog.comment.get    - Get comment by ID
  - services.blog.comment.list   - List comments by post ID
  - services.blog.comment.update - Update comment by ID
  - services.blog.comment.delete - Delete comment by ID

Use the nats CLI to interact with services:
  nats request services.article.create '{"title":"Hello","content":"World","slug":"hello-world"}'
  nats request services.blog.post.create '{"title":"My Post","content":"Content","slug":"my-post"}'

Run ./demo.sh to see full CRUD workflow

Press Ctrl+C to shutdown gracefully
`)
}
