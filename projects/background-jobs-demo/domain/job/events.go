package job

import "time"

// EventType represents the type of job event.
type EventType string

const (
	// EventTypeJobStarted indicates a job has started processing.
	EventTypeJobStarted EventType = "job.started"
	// EventTypeJobProgress indicates a job has made progress.
	EventTypeJobProgress EventType = "job.progress"
	// EventTypeJobCompleted indicates a job has completed successfully.
	EventTypeJobCompleted EventType = "job.completed"
	// EventTypeJobFailed indicates a job has failed.
	EventTypeJobFailed EventType = "job.failed"
	// EventTypeJobDeadLetter indicates a job was moved to dead-letter queue.
	EventTypeJobDeadLetter EventType = "job.dead_letter"
)

// JobEvent represents an event related to job processing.
type JobEvent struct {
	Type      EventType   `json:"type"`
	JobID     string      `json:"job_id"`
	JobType   JobType     `json:"job_type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// JobStartedData contains data for a job started event.
type JobStartedData struct {
	WorkerID string `json:"worker_id"`
}

// JobProgressData contains data for a job progress event.
type JobProgressData struct {
	Progress int    `json:"progress"`
	Message  string `json:"message,omitempty"`
}

// JobCompletedData contains data for a job completed event.
type JobCompletedData struct {
	Duration time.Duration `json:"duration"`
	Result   interface{}   `json:"result,omitempty"`
}

// JobFailedData contains data for a job failed event.
type JobFailedData struct {
	Error      string `json:"error"`
	RetryCount int    `json:"retry_count"`
	WillRetry  bool   `json:"will_retry"`
}

// JobDeadLetterData contains data for a dead-letter event.
type JobDeadLetterData struct {
	Reason     string `json:"reason"`
	RetryCount int    `json:"retry_count"`
}

// NewJobStartedEvent creates a new job started event.
func NewJobStartedEvent(jobID string, jobType JobType, workerID string) JobEvent {
	return JobEvent{
		Type:      EventTypeJobStarted,
		JobID:     jobID,
		JobType:   jobType,
		Timestamp: time.Now(),
		Data:      JobStartedData{WorkerID: workerID},
	}
}

// NewJobProgressEvent creates a new job progress event.
func NewJobProgressEvent(jobID string, jobType JobType, progress int, message string) JobEvent {
	return JobEvent{
		Type:      EventTypeJobProgress,
		JobID:     jobID,
		JobType:   jobType,
		Timestamp: time.Now(),
		Data:      JobProgressData{Progress: progress, Message: message},
	}
}

// NewJobCompletedEvent creates a new job completed event.
func NewJobCompletedEvent(jobID string, jobType JobType, duration time.Duration, result interface{}) JobEvent {
	return JobEvent{
		Type:      EventTypeJobCompleted,
		JobID:     jobID,
		JobType:   jobType,
		Timestamp: time.Now(),
		Data:      JobCompletedData{Duration: duration, Result: result},
	}
}

// NewJobFailedEvent creates a new job failed event.
func NewJobFailedEvent(jobID string, jobType JobType, err string, retryCount int, willRetry bool) JobEvent {
	return JobEvent{
		Type:      EventTypeJobFailed,
		JobID:     jobID,
		JobType:   jobType,
		Timestamp: time.Now(),
		Data:      JobFailedData{Error: err, RetryCount: retryCount, WillRetry: willRetry},
	}
}

// NewJobDeadLetterEvent creates a new dead-letter event.
func NewJobDeadLetterEvent(jobID string, jobType JobType, reason string, retryCount int) JobEvent {
	return JobEvent{
		Type:      EventTypeJobDeadLetter,
		JobID:     jobID,
		JobType:   jobType,
		Timestamp: time.Now(),
		Data:      JobDeadLetterData{Reason: reason, RetryCount: retryCount},
	}
}
