package api

import "errors"

// Errors of the job service, mapped to HTTP status codes by the handlers.
var (
	// ErrJobNotFound indicates the job was not found (404).
	ErrJobNotFound = errors.New("job not found")
	// ErrInvalidJobType indicates an invalid job type was provided (400).
	ErrInvalidJobType = errors.New("invalid job type")
	// ErrInvalidPayload indicates an invalid payload was provided (400).
	ErrInvalidPayload = errors.New("invalid payload")
	// ErrQueueUnavailable indicates the job could not be enqueued (503).
	ErrQueueUnavailable = errors.New("queue unavailable")
)

// rejection is an error carrying the worker's reason verbatim while matching
// one of the sentinel errors above with errors.Is.
type rejection struct {
	kind error
	msg  string
}

func (r *rejection) Error() string { return r.msg }
func (r *rejection) Unwrap() error { return r.kind }
