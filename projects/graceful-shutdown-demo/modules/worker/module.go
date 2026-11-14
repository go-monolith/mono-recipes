package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-monolith/mono"
)

// Config configures the background worker. Zero values use the defaults.
type Config struct {
	// Interval between tasks (default 2s).
	Interval time.Duration
	// TaskDuration is how long one simulated task takes (default 500ms).
	TaskDuration time.Duration
}

// WorkerModule implements the mono Module interface for a background worker.
type WorkerModule struct {
	interval     time.Duration
	taskDuration time.Duration

	stopChan chan struct{}
	doneChan chan struct{}
	stopOnce sync.Once
}

// Compile-time interface check
var _ mono.Module = (*WorkerModule)(nil)

// NewModule creates the background worker module.
func NewModule(cfg Config) *WorkerModule {
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.TaskDuration <= 0 {
		cfg.TaskDuration = 500 * time.Millisecond
	}
	return &WorkerModule{interval: cfg.Interval, taskDuration: cfg.TaskDuration}
}

// Name returns the module name.
func (m *WorkerModule) Name() string {
	return "worker"
}

// Start initializes and starts the background worker.
func (m *WorkerModule) Start(_ context.Context) error {
	m.stopChan = make(chan struct{})
	m.doneChan = make(chan struct{})

	go m.run()

	log.Println("Background worker started")
	return nil
}

// run executes the worker's periodic tasks.
func (m *WorkerModule) run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	defer close(m.doneChan)

	taskID := 0
	for {
		select {
		case <-m.stopChan:
			log.Println("Background worker received stop signal")
			return
		case <-ticker.C:
			taskID++
			m.performTask(taskID)
		}
	}
}

// performTask simulates a background task that can be interrupted.
func (m *WorkerModule) performTask(taskID int) {
	log.Printf("Background worker: Processing task #%d...\n", taskID)

	select {
	case <-time.After(m.taskDuration):
		log.Printf("Background worker: Task #%d completed\n", taskID)
	case <-m.stopChan:
		log.Printf("Background worker: Task #%d interrupted\n", taskID)
	}
}

// Stop gracefully shuts down the background worker. It interrupts the task in
// progress and returns once the worker goroutine has exited, or when ctx's
// deadline passes, whichever comes first.
func (m *WorkerModule) Stop(ctx context.Context) error {
	if m.stopChan == nil {
		return nil
	}

	log.Println("Shutting down background worker...")

	m.stopOnce.Do(func() {
		close(m.stopChan)
	})

	select {
	case <-m.doneChan:
		log.Println("Background worker stopped gracefully")
	case <-ctx.Done():
		log.Println("Background worker shutdown timeout exceeded")
		return ctx.Err()
	}

	return nil
}
