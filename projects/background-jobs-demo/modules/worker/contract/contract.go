// Package contract is the published port of the worker module, the single
// owner of job state: the job DTOs, the typed request-reply contracts other
// modules create and query jobs through, and the queue group contract jobs are
// processed on. It is the only package of the worker module another module may
// import.
//
// A contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import (
	"encoding/json"
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// Job is the wire view of a job record owned by the worker module.
type Job struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`   // "email", "image_processing" or "report_generation"
	Status          string          `json:"status"` // "pending", "processing", "completed", "failed", "dead_letter"
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

// Rejection codes of CreateJobResponse.ErrorCode.
const (
	ErrorCodeInvalidJobType   = "invalid_job_type"
	ErrorCodeInvalidPayload   = "invalid_payload"
	ErrorCodeQueueUnavailable = "queue_unavailable"
)

// CreateJobRequest asks the worker to record a new job and enqueue it.
type CreateJobRequest struct {
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
	Priority int             `json:"priority,omitempty"`
}

// CreateJobResponse carries the created job (status "pending"), or a
// rejection: ErrorCode is one of the ErrorCode* constants and Error the
// human-readable reason. A rejection is a response, not a transport error.
type CreateJobResponse struct {
	Job       *Job   `json:"job,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Error     string `json:"error,omitempty"`
}

// GetJobRequest asks for one job by ID.
type GetJobRequest struct {
	ID string `json:"id"`
}

// GetJobResponse carries the job, or Found=false.
type GetJobResponse struct {
	Job   *Job `json:"job,omitempty"`
	Found bool `json:"found"`
}

// ListJobsRequest filters (empty means any) and paginates the job list.
type ListJobsRequest struct {
	Status string `json:"status,omitempty"`
	Type   string `json:"type,omitempty"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// ListJobsResponse carries one page of jobs.
type ListJobsResponse struct {
	Jobs []Job `json:"jobs"`
}

// JobMessage is the payload of the process-job queue group service: the job to
// run. Its record (status, progress, result) is tracked by ID in the worker's
// job store.
type JobMessage struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Priority   int             `json:"priority"`
	MaxRetries int             `json:"max_retries"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CreateJob validates and records a job, then enqueues it on ProcessJob.
// Subject: services.worker.create-job
var CreateJob = helper.RequestReplyService[CreateJobRequest, CreateJobResponse]("worker", "create-job")

// GetJob returns a job by ID. Subject: services.worker.get-job
var GetJob = helper.RequestReplyService[GetJobRequest, GetJobResponse]("worker", "get-job")

// ListJobs lists jobs with optional filters. Subject: services.worker.list-jobs
var ListJobs = helper.RequestReplyService[ListJobsRequest, ListJobsResponse]("worker", "list-jobs")

// ProcessJob is the fire-and-forget queue group service that runs a job.
// Subject: services.worker.process-job. Send returns once the worker accepted
// the message; processing happens asynchronously.
var ProcessJob = helper.QueueGroupService[JobMessage]("worker", "process-job")
