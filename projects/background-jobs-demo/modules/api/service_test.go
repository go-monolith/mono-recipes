package api

import (
	"context"
	"testing"

	"github.com/example/background-jobs-demo/domain/job"
)

// mockNatsClient is a mock implementation of the NATS client for testing
type mockNatsClient struct {
	publishedJobs []*job.Job
	shouldFail    bool
}

func (m *mockNatsClient) PublishJob(ctx context.Context, j *job.Job) error {
	if m.shouldFail {
		return job.ErrQueueUnavailable
	}
	m.publishedJobs = append(m.publishedJobs, j)
	return nil
}

func TestService_CreateJob_Success(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	tests := []struct {
		name    string
		req     *job.CreateJobRequest
		wantErr bool
	}{
		{
			name: "valid email job",
			req: &job.CreateJobRequest{
				Type: job.JobTypeEmail,
				Payload: map[string]interface{}{
					"to":      "test@example.com",
					"subject": "Test",
					"body":    "Test body",
				},
				Priority: 1,
			},
			wantErr: false,
		},
		{
			name: "valid image processing job",
			req: &job.CreateJobRequest{
				Type: job.JobTypeImageProcessing,
				Payload: map[string]interface{}{
					"image_url":   "https://example.com/image.jpg",
					"operations":  []string{"resize"},
					"output_path": "/output/image.jpg",
				},
				Priority: 2,
			},
			wantErr: false,
		},
		{
			name: "valid report generation job",
			req: &job.CreateJobRequest{
				Type: job.JobTypeReportGeneration,
				Payload: map[string]interface{}{
					"report_type": "sales",
					"format":      "pdf",
					"date_range": map[string]interface{}{
						"start": "2024-01-01",
						"end":   "2024-01-31",
					},
				},
				Priority: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j, err := service.CreateJob(context.Background(), tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if j == nil {
					t.Error("CreateJob() returned nil job")
					return
				}

				if j.Type != tt.req.Type {
					t.Errorf("CreateJob() job type = %v, want %v", j.Type, tt.req.Type)
				}

				if j.Status != job.StatusPending {
					t.Errorf("CreateJob() job status = %v, want %v", j.Status, job.StatusPending)
				}

				// Verify job was stored
				stored, err := jobStore.GetByID(j.ID)
				if err != nil {
					t.Errorf("Job not found in store: %v", err)
				}
				if stored.ID != j.ID {
					t.Errorf("Stored job ID = %v, want %v", stored.ID, j.ID)
				}

				// Verify job was published to NATS
				if len(mockNats.publishedJobs) == 0 {
					t.Error("Job was not published to NATS")
				}
			}
		})
	}
}

func TestService_CreateJob_InvalidType(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	req := &job.CreateJobRequest{
		Type:     job.JobType("invalid_type"),
		Payload:  map[string]interface{}{},
		Priority: 0,
	}

	_, err := service.CreateJob(context.Background(), req)
	if err != job.ErrInvalidJobType {
		t.Errorf("CreateJob() error = %v, want %v", err, job.ErrInvalidJobType)
	}
}

func TestService_CreateJob_InvalidPayload(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	tests := []struct {
		name    string
		jobType job.JobType
		payload map[string]interface{}
	}{
		{
			name:    "email missing required fields",
			jobType: job.JobTypeEmail,
			payload: map[string]interface{}{
				"to": "test@example.com",
				// missing subject and body
			},
		},
		{
			name:    "image processing missing required fields",
			jobType: job.JobTypeImageProcessing,
			payload: map[string]interface{}{
				"image_url": "https://example.com/image.jpg",
				// missing output_path
			},
		},
		{
			name:    "report generation missing required fields",
			jobType: job.JobTypeReportGeneration,
			payload: map[string]interface{}{
				"report_type": "sales",
				// missing format
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &job.CreateJobRequest{
				Type:     tt.jobType,
				Payload:  tt.payload,
				Priority: 0,
			}

			_, err := service.CreateJob(context.Background(), req)
			if err != job.ErrInvalidPayload {
				t.Errorf("CreateJob() error = %v, want %v", err, job.ErrInvalidPayload)
			}
		})
	}
}

func TestService_CreateJob_QueueUnavailable(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{shouldFail: true, publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	req := &job.CreateJobRequest{
		Type: job.JobTypeEmail,
		Payload: map[string]interface{}{
			"to":      "test@example.com",
			"subject": "Test",
			"body":    "Test body",
		},
		Priority: 1,
	}

	_, err := service.CreateJob(context.Background(), req)
	if err == nil {
		t.Error("CreateJob() expected error when queue is unavailable")
	}

	// Verify job status was updated to failed
	jobs := jobStore.List("", "", 100, 0)
	if len(jobs) > 0 && jobs[0].Status != job.StatusFailed {
		t.Errorf("Job status = %v, want %v after queue failure", jobs[0].Status, job.StatusFailed)
	}
}

func TestService_GetJob(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	// Create a job
	j := job.NewJob(job.JobTypeEmail, []byte(`{"to":"test@example.com"}`), 1)
	_ = jobStore.Create(j)

	// Get the job
	retrieved, err := service.GetJob(j.ID)
	if err != nil {
		t.Errorf("GetJob() error = %v", err)
	}

	if retrieved.ID != j.ID {
		t.Errorf("GetJob() ID = %v, want %v", retrieved.ID, j.ID)
	}
}

func TestService_GetJob_NotFound(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	_, err := service.GetJob("non-existent-id")
	if err != job.ErrJobNotFound {
		t.Errorf("GetJob() error = %v, want %v", err, job.ErrJobNotFound)
	}
}

func TestService_ListJobs(t *testing.T) {
	jobStore := job.NewStore()
	mockNats := &mockNatsClient{publishedJobs: make([]*job.Job, 0)}
	service := NewService(jobStore, mockNats)

	// Create multiple jobs
	for i := 0; i < 5; i++ {
		j := job.NewJob(job.JobTypeEmail, []byte(`{"to":"test@example.com"}`), i)
		_ = jobStore.Create(j)
	}

	// List all jobs
	jobs := service.ListJobs("", "", 10, 0)
	if len(jobs) != 5 {
		t.Errorf("ListJobs() count = %v, want %v", len(jobs), 5)
	}

	// Test pagination
	jobs = service.ListJobs("", "", 2, 0)
	if len(jobs) != 2 {
		t.Errorf("ListJobs() with limit count = %v, want %v", len(jobs), 2)
	}

	jobs = service.ListJobs("", "", 2, 2)
	if len(jobs) != 2 {
		t.Errorf("ListJobs() with offset count = %v, want %v", len(jobs), 2)
	}
}
