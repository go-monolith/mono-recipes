// Package eventbus provides an in-memory event bus for job events.
package eventbus

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
)

// EventHandler is a function that handles job events.
type EventHandler func(event job.JobEvent)

// EventBus provides publish-subscribe functionality for job events.
type EventBus struct {
	handlers map[job.EventType][]EventHandler
	mu       sync.RWMutex
}

// New creates a new EventBus.
func New() *EventBus {
	return &EventBus{
		handlers: make(map[job.EventType][]EventHandler),
	}
}

// Subscribe registers a handler for a specific event type.
func (eb *EventBus) Subscribe(eventType job.EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	log.Printf("[eventbus] Subscribed to %s", eventType)
}

// SubscribeAll registers a handler for all event types.
func (eb *EventBus) SubscribeAll(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eventTypes := []job.EventType{
		job.EventTypeJobStarted,
		job.EventTypeJobProgress,
		job.EventTypeJobCompleted,
		job.EventTypeJobFailed,
		job.EventTypeJobDeadLetter,
	}

	for _, et := range eventTypes {
		eb.handlers[et] = append(eb.handlers[et], handler)
	}
	log.Println("[eventbus] Subscribed to all event types")
}

// Publish publishes an event to all registered handlers.
func (eb *EventBus) Publish(_ context.Context, event job.JobEvent) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		// Run handlers asynchronously to not block the publisher
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[eventbus] Handler panic for %s: %v", event.Type, r)
				}
			}()
			h(event)
		}(handler)
	}
}

// PublishJobStarted publishes a job started event.
func (eb *EventBus) PublishJobStarted(ctx context.Context, jobID string, jobType job.JobType, workerID string) {
	eb.Publish(ctx, job.NewJobStartedEvent(jobID, jobType, workerID))
}

// PublishJobProgress publishes a job progress event.
func (eb *EventBus) PublishJobProgress(ctx context.Context, jobID string, jobType job.JobType, progress int, message string) {
	eb.Publish(ctx, job.NewJobProgressEvent(jobID, jobType, progress, message))
}

// PublishJobCompleted publishes a job completed event.
func (eb *EventBus) PublishJobCompleted(ctx context.Context, jobID string, jobType job.JobType, duration int64, result interface{}) {
	eb.Publish(ctx, job.NewJobCompletedEvent(jobID, jobType, time.Duration(duration)*time.Millisecond, result))
}

// PublishJobFailed publishes a job failed event.
func (eb *EventBus) PublishJobFailed(ctx context.Context, jobID string, jobType job.JobType, err string, retryCount int, willRetry bool) {
	eb.Publish(ctx, job.NewJobFailedEvent(jobID, jobType, err, retryCount, willRetry))
}

// PublishJobDeadLetter publishes a job dead-letter event.
func (eb *EventBus) PublishJobDeadLetter(ctx context.Context, jobID string, jobType job.JobType, reason string, retryCount int) {
	eb.Publish(ctx, job.NewJobDeadLetterEvent(jobID, jobType, reason, retryCount))
}

// HandlerCount returns the number of handlers for a specific event type.
func (eb *EventBus) HandlerCount(eventType job.EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers[eventType])
}
