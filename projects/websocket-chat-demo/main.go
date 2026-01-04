package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/websocket-chat-demo/modules/chat"
	"github.com/example/websocket-chat-demo/modules/wsserver"
)

func main() {
	// Configuration
	httpAddr := getEnv("HTTP_ADDR", ":8080")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	logger := app.Logger()

	// Create modules
	// The chat module handles:
	// - Room management (create, list, join, leave)
	// - Message broadcasting via EventBus
	// - Message history storage
	// - Exposes services via ServiceProviderModule for inter-module communication
	chatModule := chat.NewModule(logger)

	// The WebSocket server module handles:
	// - WebSocket connections and message handling
	// - REST API for room management
	// - Broadcasting messages to connected clients via event consumption
	// - Uses DependentModule to receive chat service container
	wsServerModule := wsserver.NewModule(httpAddr, logger)

	// Register modules
	// The framework handles:
	// 1. Dependency injection (wsserver depends on chat, receives ServiceContainer)
	// 2. EventBus wiring (chat emits events, wsserver consumes for broadcasting)
	// 3. Service registration (chat registers Request-Reply services)
	// 4. Lifecycle management (Start/Stop in correct order based on dependencies)
	app.Register(chatModule)
	app.Register(wsServerModule)

	// Start the application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Printf("WebSocket Chat Demo started on %s", httpAddr)
	log.Println("WebSocket endpoint: ws://localhost" + httpAddr + "/ws")
	log.Println("Press Ctrl+C to shutdown...")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}

	log.Println("Application stopped successfully")
}

// getEnv returns the environment variable value or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
