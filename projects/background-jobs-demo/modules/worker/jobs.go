package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-monolith/mono"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/worker/contract"
)

// createJob validates and records a job, then enqueues it on the module's own
// process-job queue group. Rejections are returned as response codes.
func (m *Module) createJob(ctx context.Context, req contract.CreateJobRequest, _ *mono.Msg) (contract.CreateJobResponse, error) {
	jobType := job.JobType(req.Type)
	if !jobType.IsValid() {
		return contract.CreateJobResponse{ErrorCode: contract.ErrorCodeInvalidJobType, Error: job.ErrInvalidJobType.Error()}, nil
	}
	if err := validatePayload(jobType, req.Payload); err != nil {
		return contract.CreateJobResponse{ErrorCode: contract.ErrorCodeInvalidPayload, Error: err.Error()}, nil
	}

	payload := req.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("null")
	}
	j := job.NewJob(jobType, payload, req.Priority)
	if err := m.jobStore.Create(j); err != nil {
		return contract.CreateJobResponse{}, fmt.Errorf("failed to store job: %w", err)
	}
	created := toContractJob(j) // snapshot before processing starts

	msg := contract.JobMessage{
		ID:         j.ID,
		Type:       string(j.Type),
		Payload:    j.Payload,
		Priority:   j.Priority,
		MaxRetries: j.MaxRetries,
		CreatedAt:  j.CreatedAt,
	}
	if err := contract.ProcessJob.Send(ctx, m.container, msg); err != nil {
		_ = m.jobStore.UpdateStatus(j.ID, job.JobStatusFailed)
		log.Printf("[worker] Failed to enqueue job %s: %v", j.ID, err)
		return contract.CreateJobResponse{ErrorCode: contract.ErrorCodeQueueUnavailable, Error: job.ErrQueueUnavailable.Error()}, nil
	}

	return contract.CreateJobResponse{Job: &created}, nil
}

// getJob returns a job by ID.
func (m *Module) getJob(_ context.Context, req contract.GetJobRequest, _ *mono.Msg) (contract.GetJobResponse, error) {
	j, err := m.jobStore.GetByID(req.ID)
	if err != nil {
		return contract.GetJobResponse{Found: false}, nil
	}
	resp := toContractJob(j)
	return contract.GetJobResponse{Job: &resp, Found: true}, nil
}

// listJobs returns one page of jobs matching the filters.
func (m *Module) listJobs(_ context.Context, req contract.ListJobsRequest, _ *mono.Msg) (contract.ListJobsResponse, error) {
	jobs := m.jobStore.List(job.JobStatus(req.Status), job.JobType(req.Type), req.Limit, req.Offset)
	resp := contract.ListJobsResponse{Jobs: make([]contract.Job, 0, len(jobs))}
	for _, j := range jobs {
		resp.Jobs = append(resp.Jobs, toContractJob(j))
	}
	return resp, nil
}

// validatePayload checks the payload against the job type's payload schema.
func validatePayload(jobType job.JobType, payload json.RawMessage) error {
	data := []byte(payload)
	if len(data) == 0 {
		data = []byte("null")
	}

	switch jobType {
	case job.JobTypeEmail:
		var p job.EmailPayload
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("%w: invalid email payload", job.ErrInvalidPayload)
		}
		if p.To == "" || p.Subject == "" || p.Body == "" {
			return fmt.Errorf("%w: email payload requires to, subject, and body", job.ErrInvalidPayload)
		}

	case job.JobTypeImageProcessing:
		var p job.ImageProcessingPayload
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("%w: invalid image processing payload", job.ErrInvalidPayload)
		}
		if p.ImageURL == "" || p.OutputPath == "" {
			return fmt.Errorf("%w: image processing payload requires image_url and output_path", job.ErrInvalidPayload)
		}

	case job.JobTypeReportGeneration:
		var p job.ReportGenerationPayload
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("%w: invalid report generation payload", job.ErrInvalidPayload)
		}
		if p.ReportType == "" || p.Format == "" {
			return fmt.Errorf("%w: report generation payload requires report_type and format", job.ErrInvalidPayload)
		}

	default:
		return job.ErrInvalidJobType
	}
	return nil
}

// toContractJob converts a domain job to its wire view.
func toContractJob(j *job.Job) contract.Job {
	return contract.Job{
		ID:              j.ID,
		Type:            string(j.Type),
		Status:          string(j.Status),
		Payload:         j.Payload,
		Result:          j.Result,
		Error:           j.Error,
		Priority:        j.Priority,
		Progress:        j.Progress,
		ProgressMessage: j.ProgressMessage,
		RetryCount:      j.RetryCount,
		MaxRetries:      j.MaxRetries,
		WorkerID:        j.WorkerID,
		CreatedAt:       j.CreatedAt,
		StartedAt:       j.StartedAt,
		CompletedAt:     j.CompletedAt,
		UpdatedAt:       j.UpdatedAt,
	}
}
