// Package api provides REST API handlers for job management.
package api

import (
	"context"
	"encoding/json"
	"fmt"

	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// Service maps the HTTP use cases onto the worker's job contracts.
type Service struct {
	jobs JobsPort
}

// NewService creates a new job service over the worker's job contracts.
func NewService(jobs JobsPort) *Service {
	return &Service{jobs: jobs}
}

// CreateJob asks the worker to record and enqueue a new job.
func (s *Service) CreateJob(ctx context.Context, jobType string, payload any, priority int) (*workercontract.Job, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}

	resp, err := s.jobs.CreateJob(ctx, workercontract.CreateJobRequest{Type: jobType, Payload: data, Priority: priority})
	if err != nil {
		// The worker could not be reached: nothing was enqueued.
		return nil, fmt.Errorf("%w: %v", ErrQueueUnavailable, err)
	}
	switch resp.ErrorCode {
	case "":
	case workercontract.ErrorCodeInvalidJobType:
		return nil, &rejection{kind: ErrInvalidJobType, msg: resp.Error}
	case workercontract.ErrorCodeInvalidPayload:
		return nil, &rejection{kind: ErrInvalidPayload, msg: resp.Error}
	case workercontract.ErrorCodeQueueUnavailable:
		return nil, &rejection{kind: ErrQueueUnavailable, msg: resp.Error}
	default:
		return nil, fmt.Errorf("create job rejected: %s: %s", resp.ErrorCode, resp.Error)
	}
	if resp.Job == nil {
		return nil, fmt.Errorf("create job: worker returned no job")
	}
	return resp.Job, nil
}

// GetJob retrieves a job by ID.
func (s *Service) GetJob(ctx context.Context, id string) (*workercontract.Job, error) {
	resp, err := s.jobs.GetJob(ctx, workercontract.GetJobRequest{ID: id})
	if err != nil {
		return nil, err
	}
	if !resp.Found || resp.Job == nil {
		return nil, ErrJobNotFound
	}
	return resp.Job, nil
}

// ListJobs returns one page of jobs with optional filtering.
func (s *Service) ListJobs(ctx context.Context, status, jobType string, limit, offset int) ([]workercontract.Job, error) {
	resp, err := s.jobs.ListJobs(ctx, workercontract.ListJobsRequest{Status: status, Type: jobType, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return resp.Jobs, nil
}
