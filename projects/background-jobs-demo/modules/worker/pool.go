// Package worker provides a worker pool for processing background jobs.
package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/eventbus"
	"github.com/example/background-jobs-demo/modules/nats"
)

// PoolConfig holds worker pool configuration.
type PoolConfig struct {
	NumWorkers      int
	MaxRetries      int
	BaseRetryDelay  time.Duration
	MaxRetryDelay   time.Duration
	ProcessTimeout  time.Duration
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		NumWorkers:      3,
		MaxRetries:      5,
		BaseRetryDelay:  time.Second,
		MaxRetryDelay:   time.Minute,
		ProcessTimeout:  5 * time.Minute,
	}
}

// Pool manages a pool of workers for processing jobs.
type Pool struct {
	config     PoolConfig
	natsClient *nats.Client
	eventBus   *eventbus.EventBus
	jobStore   *job.Store
	workers    []*Worker
	wg         sync.WaitGroup
	cancel     context.CancelFunc
	mu         sync.RWMutex
	running    bool
}

// NewPool creates a new worker pool.
func NewPool(cfg PoolConfig, natsClient *nats.Client, eventBus *eventbus.EventBus, jobStore *job.Store) *Pool {
	return &Pool{
		config:     cfg,
		natsClient: natsClient,
		eventBus:   eventBus,
		jobStore:   jobStore,
		workers:    make([]*Worker, 0, cfg.NumWorkers),
	}
}

// Start starts the worker pool and begins processing jobs.
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pool is already running")
	}
	p.running = true
	p.mu.Unlock()

	// Subscribe to job messages
	msgChan, err := p.natsClient.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to jobs: %w", err)
	}

	// Create cancellable context for workers
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	// Start workers
	for i := 0; i < p.config.NumWorkers; i++ {
		workerID := fmt.Sprintf("worker-%d", i+1)
		worker := NewWorker(workerID, p.config, p.eventBus, p.jobStore, p.natsClient)
		p.workers = append(p.workers, worker)

		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.Run(workerCtx, msgChan)
		}(worker)

		log.Printf("[pool] Started %s", workerID)
	}

	log.Printf("[pool] Worker pool started with %d workers", p.config.NumWorkers)
	return nil
}

// Stop stops the worker pool gracefully.
func (p *Pool) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	// Cancel worker context
	if p.cancel != nil {
		p.cancel()
	}

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[pool] All workers stopped gracefully")
	case <-ctx.Done():
		log.Println("[pool] Timeout waiting for workers to stop")
		return ctx.Err()
	}

	return nil
}

// IsRunning returns true if the pool is running.
func (p *Pool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Worker processes jobs from the message channel.
type Worker struct {
	id         string
	config     PoolConfig
	processor  *Processor
	eventBus   *eventbus.EventBus
	jobStore   *job.Store
	natsClient *nats.Client
}

// NewWorker creates a new worker.
func NewWorker(id string, cfg PoolConfig, eventBus *eventbus.EventBus, jobStore *job.Store, natsClient *nats.Client) *Worker {
	return &Worker{
		id:         id,
		config:     cfg,
		processor:  NewProcessor(id),
		eventBus:   eventBus,
		jobStore:   jobStore,
		natsClient: natsClient,
	}
}

// Run starts the worker's main processing loop.
func (w *Worker) Run(ctx context.Context, msgChan <-chan *nats.ConsumeMessage) {
	log.Printf("[%s] Worker started", w.id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Worker stopping due to context cancellation", w.id)
			return
		case msg, ok := <-msgChan:
			if !ok {
				log.Printf("[%s] Message channel closed, worker stopping", w.id)
				return
			}
			w.processMessage(ctx, msg)
		}
	}
}

// processMessage processes a single job message.
func (w *Worker) processMessage(ctx context.Context, msg *nats.ConsumeMessage) {
	j := msg.Job
	deliveryCount := msg.DeliveryCount

	log.Printf("[%s] Processing job %s (type=%s, delivery=%d)", w.id, j.ID, j.Type, deliveryCount)

	// Update job status in store
	if err := w.jobStore.SetStarted(j.ID, w.id); err != nil {
		log.Printf("[%s] Error updating job status: %v", w.id, err)
	}

	// Publish job started event
	w.eventBus.PublishJobStarted(ctx, j.ID, j.Type, w.id)

	// Create progress callback
	progressFn := func(progress int, message string) {
		if err := w.jobStore.UpdateProgress(j.ID, progress, message); err != nil {
			log.Printf("[%s] Error updating progress: %v", w.id, err)
		}
		w.eventBus.PublishJobProgress(ctx, j.ID, j.Type, progress, message)
	}

	// Create processing context with timeout
	processCtx, cancel := context.WithTimeout(ctx, w.config.ProcessTimeout)
	defer cancel()

	// Process the job
	startTime := time.Now()
	result, err := w.processor.Process(processCtx, j, progressFn)
	duration := time.Since(startTime)

	// Handle processing errors (context cancellation, timeout, etc.)
	if err != nil {
		log.Printf("[%s] Job %s processing error: %v", w.id, j.ID, err)
		w.handleFailure(ctx, msg, j, fmt.Sprintf("processing error: %v", err), deliveryCount)
		return
	}

	// Handle processing result
	if result.Success {
		w.handleSuccess(ctx, msg, j, result, duration)
	} else {
		errMsg := "unknown error"
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		w.handleFailure(ctx, msg, j, errMsg, deliveryCount)
	}
}

// handleSuccess handles successful job processing.
func (w *Worker) handleSuccess(ctx context.Context, msg *nats.ConsumeMessage, j *job.Job, result *ProcessResult, duration time.Duration) {
	// Update job status
	if err := w.jobStore.SetCompleted(j.ID, result.Result); err != nil {
		log.Printf("[%s] Error setting job completed: %v", w.id, err)
	}

	// Acknowledge message
	if err := msg.Ack(); err != nil {
		log.Printf("[%s] Error acknowledging message: %v", w.id, err)
	}

	// Publish completion event
	w.eventBus.PublishJobCompleted(ctx, j.ID, j.Type, duration.Milliseconds(), result.Result)

	log.Printf("[%s] Job %s completed successfully in %v", w.id, j.ID, duration)
}

// handleFailure handles failed job processing.
func (w *Worker) handleFailure(ctx context.Context, msg *nats.ConsumeMessage, j *job.Job, errMsg string, deliveryCount int) {
	// Increment retry count in store
	newRetryCount, err := w.jobStore.IncrementRetry(j.ID)
	if err != nil {
		log.Printf("[%s] Error incrementing retry count: %v", w.id, err)
		newRetryCount = deliveryCount
	}

	// Check if max retries exceeded
	if newRetryCount >= w.config.MaxRetries {
		w.moveToDeadLetter(ctx, msg, j, errMsg, newRetryCount)
		return
	}

	// Calculate retry delay with exponential backoff
	delay := w.calculateRetryDelay(newRetryCount)

	// Update job status
	if err := w.jobStore.SetFailed(j.ID, errMsg); err != nil {
		log.Printf("[%s] Error setting job failed: %v", w.id, err)
	}

	// Negative acknowledge with delay for retry
	if err := msg.NakWithDelay(delay); err != nil {
		log.Printf("[%s] Error NAK with delay: %v", w.id, err)
	}

	// Publish failure event
	w.eventBus.PublishJobFailed(ctx, j.ID, j.Type, errMsg, newRetryCount, true)

	log.Printf("[%s] Job %s failed (retry %d/%d), will retry in %v: %s",
		w.id, j.ID, newRetryCount, w.config.MaxRetries, delay, errMsg)
}

// moveToDeadLetter moves a job to the dead-letter queue.
func (w *Worker) moveToDeadLetter(ctx context.Context, msg *nats.ConsumeMessage, j *job.Job, errMsg string, retryCount int) {
	reason := fmt.Sprintf("max retries (%d) exceeded: %s", w.config.MaxRetries, errMsg)

	// Update job status in store
	if err := w.jobStore.SetDeadLetter(j.ID, reason); err != nil {
		log.Printf("[%s] Error setting job dead letter: %v", w.id, err)
	}

	// Publish to dead-letter queue
	if err := w.natsClient.PublishDeadLetter(ctx, j, reason); err != nil {
		log.Printf("[%s] Error publishing to dead-letter queue: %v", w.id, err)
	}

	// Terminate the message (no more retries)
	if err := msg.Term(); err != nil {
		log.Printf("[%s] Error terminating message: %v", w.id, err)
	}

	// Publish dead-letter event
	w.eventBus.PublishJobDeadLetter(ctx, j.ID, j.Type, reason, retryCount)

	log.Printf("[%s] Job %s moved to dead-letter queue: %s", w.id, j.ID, reason)
}

// calculateRetryDelay calculates the delay before the next retry using exponential backoff.
func (w *Worker) calculateRetryDelay(retryCount int) time.Duration {
	// Exponential backoff: baseDelay * 2^(retryCount-1)
	delay := float64(w.config.BaseRetryDelay) * math.Pow(2, float64(retryCount-1))

	// Cap at max delay
	if time.Duration(delay) > w.config.MaxRetryDelay {
		return w.config.MaxRetryDelay
	}

	return time.Duration(delay)
}
