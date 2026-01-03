package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/websocket-chat-demo/modules/api"
	"github.com/example/websocket-chat-demo/modules/broadcast"
	"github.com/example/websocket-chat-demo/modules/chat"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== WebSocket Chat Demo - Fiber + EventBus Pubsub ===")

	// Create mono application
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create modules
	chatModule := chat.NewModule()
	broadcastModule := broadcast.NewModule()
	apiModule := api.NewModule()

	// Inject broadcast hub into API module
	// (This is done manually because the hub is not exposed via ServiceContainer)
	apiModule.SetHub(broadcastModule.GetHub())

	// Register modules with the framework.
	// Order: independent modules first, then modules with dependencies
	// - chat: Core domain (ServiceProviderModule + EventEmitterModule)
	// - broadcast: Event consumer (EventConsumerModule for WebSocket broadcasting)
	// - api: Driving adapter (Fiber HTTP/WebSocket server, depends on chat)
	app.Register(chatModule)      // Chat service + event emitter
	app.Register(broadcastModule) // WebSocket hub + event consumer
	app.Register(apiModule)       // HTTP/WebSocket API

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	log.Println("")
	log.Println("Application started successfully!")
	log.Println("")
	log.Println("Architecture:")
	log.Println("  - HTTP Framework: Fiber with WebSocket support")
	log.Println("  - Event Bus: NATS JetStream (internal pubsub)")
	log.Printf("  - NATS URL: %s", natsURL)
	log.Println("")
	log.Println("Event-Driven Chat:")
	log.Println("  - MessageSent events -> broadcast module -> WebSocket clients")
	log.Println("  - UserJoined events -> broadcast module -> WebSocket clients")
	log.Println("  - UserLeft events -> broadcast module -> WebSocket clients")
	log.Println("")
	log.Printf("REST API Endpoints (http://localhost:%s):", port)
	log.Println("  GET    /health                 - Health check")
	log.Println("  GET    /api/v1/rooms           - List all rooms")
	log.Println("  POST   /api/v1/rooms           - Create a new room")
	log.Println("  GET    /api/v1/rooms/:id       - Get room details")
	log.Println("  GET    /api/v1/rooms/:id/history - Get message history")
	log.Println("")
	log.Printf("WebSocket Endpoint (ws://localhost:%s/ws):", port)
	log.Println("  Connect with: ws://localhost:3000/ws?username=yourname")
	log.Println("  Message types: join, leave, message, history, members, room_list")
	log.Println("")
	log.Println("Example: see demo.py for Python WebSocket client demonstration")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
