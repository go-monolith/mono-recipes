package job

import "errors"

var (
	// ErrJobNotFound indicates the job was not found.
	ErrJobNotFound = errors.New("job not found")
	// ErrInvalidJobType indicates an invalid job type was provided.
	ErrInvalidJobType = errors.New("invalid job type")
	// ErrInvalidPayload indicates an invalid payload was provided.
	ErrInvalidPayload = errors.New("invalid payload")
	// ErrJobAlreadyProcessing indicates the job is already being processed.
	ErrJobAlreadyProcessing = errors.New("job already processing")
	// ErrMaxRetriesExceeded indicates the job has exceeded max retries.
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	// ErrQueueUnavailable indicates the queue is not available.
	ErrQueueUnavailable = errors.New("queue unavailable")
)
