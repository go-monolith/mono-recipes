package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/math"
	mathcontract "github.com/example/python-nats-client-demo/modules/math/contract"
	"github.com/example/python-nats-client-demo/modules/notification"
	notificationcontract "github.com/example/python-nats-client-demo/modules/notification/contract"
	"github.com/example/python-nats-client-demo/modules/payment"
	paymentcontract "github.com/example/python-nats-client-demo/modules/payment/contract"
)

const shutdownTimeout = 30 * time.Second

func main() {
	log.Println("=== Python NATS Client Demo ===")
	log.Println("Multi-service Mono application with RequestReply, QueueGroup, and StreamConsumer patterns")

	// JetStream storage directory for StreamConsumerService
	jsDir := "/tmp/python-nats-client-demo"
	// Embedded NATS port; the Python clients default to nats://localhost:4222
	// (use demo.py --nats-url when changing it).
	natsPort := getEnvInt("NATS_PORT", 4222)

	// Create mono application with JetStream enabled for StreamConsumerService
	app, err := mono.NewMonoApplication(
		mono.WithShutdownTimeout(shutdownTimeout),
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithJetStreamStorageDir(jsDir), // Required for StreamConsumerService
		mono.WithNATSPort(natsPort),
		// Reject module boundary violations (see "Module boundaries" in README).
		mono.WithStrictModuleBoundaries(),
	)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Register modules demonstrating different service patterns
	for _, m := range []mono.Module{
		math.NewModule(),         // RequestReplyService
		notification.NewModule(), // QueueGroupService
		payment.NewModule(),      // StreamConsumerService + RequestReplyService
	} {
		if err := app.Register(m); err != nil {
			log.Fatalf("Failed to register module %s: %v", m.Name(), err)
		}
	}

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Printf("NATS available at nats://localhost:%d", natsPort)
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
	log.Printf("   - %s - Math operations with response", mathcontract.Calculate.Subject())
	log.Printf("   - %s - Query payment status", paymentcontract.Status.Subject())
	log.Println("")
	log.Println("2. QueueGroupService (fire-and-forget):")
	log.Printf("   - %s - Send emails to queue", notificationcontract.SendEmail.Subject())
	log.Println("")
	log.Println("3. StreamConsumerService (durable processing):")
	log.Printf("   - %s - Process payments via JetStream", paymentcontract.ProcessPayment.Subject())
	log.Println("")
	log.Println("Run the Python demo:")
	log.Println("  python demo.py               # Full demo")
	log.Println("  python demo.py --math-only   # Math operations only")
	log.Println("  python demo.py --email-only  # Email queue only")
	log.Println("  python demo.py --payment-only # Payment stream only")
	log.Println("")
	log.Println("Press Ctrl+C to shutdown gracefully")
}

// getEnvInt returns an environment variable as int, or defaultValue.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid int value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}
