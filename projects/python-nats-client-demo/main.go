package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/python-nats-client-demo/modules/math"
	"github.com/example/python-nats-client-demo/modules/notification"
	"github.com/example/python-nats-client-demo/modules/payment"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== Python NATS Client Demo ===")
	log.Println("Multi-service Mono application with RequestReply, QueueGroup, and StreamConsumer patterns")

	// JetStream storage directory for StreamConsumerService
	jsDir := "/tmp/python-nats-client-demo"

	// Create mono application with JetStream enabled for StreamConsumerService
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithJetStreamStorageDir(jsDir), // Required for StreamConsumerService
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register modules demonstrating different service patterns
	app.Register(math.NewModule())         // RequestReplyService
	app.Register(notification.NewModule()) // QueueGroupService
	app.Register(payment.NewModule())      // StreamConsumerService + RequestReplyService

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
	log.Println("This demo shows three service patterns for Python clients:")
	log.Println("")
	log.Println("1. RequestReplyService (synchronous):")
	log.Println("   - services.math.calculate - Math operations with response")
	log.Println("   - services.payment.status - Query payment status")
	log.Println("")
	log.Println("2. QueueGroupService (fire-and-forget):")
	log.Println("   - services.notification.email-send - Send emails to queue")
	log.Println("")
	log.Println("3. StreamConsumerService (durable processing):")
	log.Println("   - services.payment.payment-process - Process payments via JetStream")
	log.Println("")
	log.Println("Run the Python demo:")
	log.Println("  python demo.py               # Full demo")
	log.Println("  python demo.py --math-only   # Math operations only")
	log.Println("  python demo.py --email-only  # Email queue only")
	log.Println("  python demo.py --payment-only # Payment stream only")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}
