// Package job provides domain types for background job processing.
package job

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JobType represents the type of background job.
type JobType string

const (
	// JobTypeEmail represents an email sending job.
	JobTypeEmail JobType = "email"
	// JobTypeImageProcessing represents an image processing job.
	JobTypeImageProcessing JobType = "image_processing"
	// JobTypeReportGeneration represents a report generation job.
	JobTypeReportGeneration JobType = "report_generation"
)

// JobStatus represents the current status of a job.
type JobStatus string

const (
	// JobStatusPending indicates the job is waiting to be processed.
	JobStatusPending JobStatus = "pending"
	// JobStatusProcessing indicates the job is currently being processed.
	JobStatusProcessing JobStatus = "processing"
	// JobStatusCompleted indicates the job has been successfully completed.
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job has failed.
	JobStatusFailed JobStatus = "failed"
	// JobStatusDeadLetter indicates the job has exceeded max retries.
	JobStatusDeadLetter JobStatus = "dead_letter"
)

// Job represents a background job to be processed.
type Job struct {
	ID              string          `json:"id"`
	Type            JobType         `json:"type"`
	Status          JobStatus       `json:"status"`
	Payload         json.RawMessage `json:"payload"`
	Result          json.RawMessage `json:"result,omitempty"`
	Error           string          `json:"error,omitempty"`
	Priority        int             `json:"priority"`
	Progress        int             `json:"progress"`
	ProgressMessage string          `json:"progress_message,omitempty"`
	RetryCount      int             `json:"retry_count"`
	MaxRetries      int             `json:"max_retries"`
	WorkerID        string          `json:"worker_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// EmailPayload represents the payload for an email job.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// ImageProcessingPayload represents the payload for an image processing job.
type ImageProcessingPayload struct {
	ImageURL   string   `json:"image_url"`
	Operations []string `json:"operations"` // resize, crop, filter, etc.
	OutputPath string   `json:"output_path"`
}

// ReportGenerationPayload represents the payload for a report generation job.
type ReportGenerationPayload struct {
	ReportType string `json:"report_type"`
	StartDate  string `json:"start_date"` // ISO 8601 date string (e.g., "2024-01-01")
	EndDate    string `json:"end_date"`   // ISO 8601 date string (e.g., "2024-01-31")
	Format     string `json:"format"`     // pdf, csv, xlsx
}

// CreateJobRequest represents a request to create a new job.
type CreateJobRequest struct {
	Type       JobType `json:"type" validate:"required"`
	Payload    any     `json:"payload" validate:"required"`
	Priority   int     `json:"priority,omitempty"`
	MaxRetries int     `json:"max_retries,omitempty"`
}

// JobResponse represents the API response for a job.
type JobResponse struct {
	Job     *Job   `json:"job"`
	Message string `json:"message,omitempty"`
}

// JobListResponse represents the API response for a list of jobs.
type JobListResponse struct {
	Jobs   []Job  `json:"jobs"`
	Total  int    `json:"total"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

// JobMessage represents a job message in the queue.
type JobMessage struct {
	Job       *Job   `json:"job"`
	MessageID string `json:"message_id"`
}

// Validate validates the CreateJobRequest.
func (r *CreateJobRequest) Validate() error {
	if r.Type == "" {
		return ErrInvalidJobType
	}
	if r.Payload == nil {
		return ErrInvalidPayload
	}
	switch r.Type {
	case JobTypeEmail, JobTypeImageProcessing, JobTypeReportGeneration:
		// Valid types
	default:
		return ErrInvalidJobType
	}
	return nil
}

// DefaultMaxRetries returns the default max retries for a job type.
func DefaultMaxRetries(jobType JobType) int {
	switch jobType {
	case JobTypeEmail:
		return 3
	case JobTypeImageProcessing:
		return 2
	case JobTypeReportGeneration:
		return 1
	default:
		return 3
	}
}

// IsValid returns true if the job type is valid.
func (jt JobType) IsValid() bool {
	switch jt {
	case JobTypeEmail, JobTypeImageProcessing, JobTypeReportGeneration:
		return true
	default:
		return false
	}
}

// NewJob creates a new job with the given type, payload, and priority.
func NewJob(jobType JobType, payload json.RawMessage, priority int) *Job {
	now := time.Now()
	return &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Status:     JobStatusPending,
		Payload:    payload,
		Priority:   priority,
		Progress:   0,
		RetryCount: 0,
		MaxRetries: DefaultMaxRetries(jobType),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewJobWithPayload creates a new job with the given type and payload struct.
func NewJobWithPayload(jobType JobType, payload any, priority int) *Job {
	data, _ := json.Marshal(payload)
	return NewJob(jobType, data, priority)
}
