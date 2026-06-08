package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/example/cron-scheduler-demo/modules/api"
	"github.com/example/cron-scheduler-demo/modules/scheduler"
	"github.com/example/cron-scheduler-demo/stats"
	gfshutdown "github.com/gelmium/graceful-shutdown"
	"github.com/go-monolith/mono"
)

const shutdownTimeout = 10 * time.Second

func main() {
	log.Println("=== Mono Cron Scheduler Demo ===")

	// CronServices are backed by the NATS JetStream message scheduler, so the
	// embedded server must run with JetStream enabled — that requires a storage
	// directory. Without WithJetStreamStorageDir, RegisterCronService fails.
	// The per-PID subdirectory keeps two instances started on the same machine
	// (common during local dev) from fighting over the same JetStream state.
	storageDir := filepath.Join(os.TempDir(), "mono-cron-scheduler-demo",
		fmt.Sprintf("run-%d", os.Getpid()))

	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithJetStreamStorageDir(storageDir),
		mono.WithShutdownTimeout(shutdownTimeout),
	)
	if err != nil {
		log.Fatalf("Failed to create mono application: %v", err)
	}

	// Shared store: the scheduler records occurrences, the api serves them.
	store := stats.NewStore()

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	if err := app.Register(scheduler.NewModule(store)); err != nil {
		log.Fatalf("Failed to register scheduler module: %v", err)
	}
	if err := app.Register(api.NewModule(store, httpAddr)); err != nil {
		log.Fatalf("Failed to register api module: %v", err)
	}

	if err := app.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	log.Printf("Application started. The 'heartbeat' cron fires every 2s.")
	log.Printf("Observe activity:  curl http://localhost%s/stats", httpAddr)
	log.Println("Press Ctrl+C to trigger graceful shutdown")

	// Wait for OS signals (SIGINT, SIGTERM) and perform graceful shutdown.
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
