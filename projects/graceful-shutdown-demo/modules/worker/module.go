package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-monolith/mono"
)

// WorkerModule implements the mono Module interface for a background worker.
type WorkerModule struct {
	stopChan chan struct{}
	doneChan chan struct{}
	stopOnce sync.Once
}

// Compile-time interface check
var _ mono.Module = (*WorkerModule)(nil)

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
	ticker := time.NewTicker(2 * time.Second)
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
	case <-time.After(500 * time.Millisecond):
		log.Printf("Background worker: Task #%d completed\n", taskID)
	case <-m.stopChan:
		log.Printf("Background worker: Task #%d interrupted\n", taskID)
	}
}

// Stop gracefully shuts down the background worker.
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
