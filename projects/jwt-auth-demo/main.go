package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/jwt-auth-demo/modules/api"
	"github.com/example/jwt-auth-demo/modules/auth"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== JWT Authentication Demo ===")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register modules with the framework
	// Order: independent modules first, then dependent modules
	app.Register(auth.NewModule()) // Independent module (provides auth services)
	app.Register(api.NewModule())  // Depends on auth module

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
	log.Println("")
	log.Println("Application started successfully!")
	log.Println("")
	log.Println("JWT Authentication Patterns Demonstrated:")
	log.Println("  - Secure password hashing with bcrypt")
	log.Println("  - JWT token generation and validation")
	log.Println("  - Access token + refresh token strategy")
	log.Println("  - Authentication middleware for protected routes")
	log.Println("  - SQLite database with GORM for user storage")
	log.Println("")
	log.Println("REST API Endpoints (http://localhost:3000):")
	log.Println("")
	log.Println("  Public Endpoints:")
	log.Println("  POST   /api/v1/auth/register  - Register a new user")
	log.Println("  POST   /api/v1/auth/login     - Login and get tokens")
	log.Println("  POST   /api/v1/auth/refresh   - Refresh access token")
	log.Println("  GET    /health                - Health check")
	log.Println("")
	log.Println("  Protected Endpoints (require Bearer token):")
	log.Println("  GET    /api/v1/profile        - Get current user profile")
	log.Println("")
	log.Println("Example: see demo.sh for curl commands to interact with the API")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
