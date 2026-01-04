// Package api provides REST API handlers for job management.
package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/worker"
	"github.com/go-monolith/mono"
)

// Service handles job-related business logic.
type Service struct {
	jobStore        *job.Store
	workerContainer mono.ServiceContainer
}

// NewService creates a new job service.
func NewService(jobStore *job.Store, workerContainer mono.ServiceContainer) *Service {
	return &Service{
		jobStore:        jobStore,
		workerContainer: workerContainer,
	}
}

// CreateJob creates a new job and enqueues it for processing.
func (s *Service) CreateJob(ctx context.Context, req *job.CreateJobRequest) (*job.Job, error) {
	// Validate job type
	if !req.Type.IsValid() {
		return nil, job.ErrInvalidJobType
	}

	// Validate and marshal payload based on job type
	payload, err := s.validateAndMarshalPayload(req.Type, req.Payload)
	if err != nil {
		return nil, err
	}

	// Create the job
	j := job.NewJob(req.Type, payload, req.Priority)

	// Store the job
	if err := s.jobStore.Create(j); err != nil {
		return nil, fmt.Errorf("failed to store job: %w", err)
	}

	// Get the QueueGroupService client
	client, err := s.workerContainer.GetQueueGroupService(worker.ServiceName)
	if err != nil {
		_ = s.jobStore.UpdateStatus(j.ID, job.JobStatusFailed)
		return nil, job.ErrQueueUnavailable
	}

	// Marshal the job for the queue
	jobData, err := json.Marshal(j)
	if err != nil {
		_ = s.jobStore.UpdateStatus(j.ID, job.JobStatusFailed)
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	// Send to worker queue (fire-and-forget)
	if err := client.Send(ctx, jobData); err != nil {
		_ = s.jobStore.UpdateStatus(j.ID, job.JobStatusFailed)
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return j, nil
}

// GetJob retrieves a job by ID.
func (s *Service) GetJob(id string) (*job.Job, error) {
	return s.jobStore.GetByID(id)
}

// ListJobs returns all jobs with optional filtering.
func (s *Service) ListJobs(status job.JobStatus, jobType job.JobType, limit, offset int) []*job.Job {
	return s.jobStore.List(status, jobType, limit, offset)
}

// validateAndMarshalPayload validates the payload for the given job type and returns JSON bytes.
func (s *Service) validateAndMarshalPayload(jobType job.JobType, payload any) (json.RawMessage, error) {
	// Marshal to JSON first
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", job.ErrInvalidPayload, err)
	}

	// Validate by unmarshaling into the correct type
	switch jobType {
	case job.JobTypeEmail:
		var p job.EmailPayload
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("%w: invalid email payload", job.ErrInvalidPayload)
		}
		if p.To == "" || p.Subject == "" || p.Body == "" {
			return nil, fmt.Errorf("%w: email payload requires to, subject, and body", job.ErrInvalidPayload)
		}

	case job.JobTypeImageProcessing:
		var p job.ImageProcessingPayload
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("%w: invalid image processing payload", job.ErrInvalidPayload)
		}
		if p.ImageURL == "" || p.OutputPath == "" {
			return nil, fmt.Errorf("%w: image processing payload requires image_url and output_path", job.ErrInvalidPayload)
		}

	case job.JobTypeReportGeneration:
		var p job.ReportGenerationPayload
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("%w: invalid report generation payload", job.ErrInvalidPayload)
		}
		if p.ReportType == "" || p.Format == "" {
			return nil, fmt.Errorf("%w: report generation payload requires report_type and format", job.ErrInvalidPayload)
		}

	default:
		return nil, job.ErrInvalidJobType
	}

	return data, nil
}
