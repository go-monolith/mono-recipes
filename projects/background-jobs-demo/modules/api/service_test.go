package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/worker"
	"github.com/go-monolith/mono"
)

// mockQueueGroupServiceClient is a mock implementation of mono.QueueGroupServiceClient
type mockQueueGroupServiceClient struct {
	sentData   [][]byte
	shouldFail bool
}

func (m *mockQueueGroupServiceClient) Send(_ context.Context, data []byte) error {
	if m.shouldFail {
		return errors.New("queue unavailable")
	}
	m.sentData = append(m.sentData, data)
	return nil
}

func (m *mockQueueGroupServiceClient) SendMsg(_ context.Context, _ *mono.Msg) error {
	if m.shouldFail {
		return errors.New("queue unavailable")
	}
	return nil
}

// mockServiceContainer is a mock implementation of mono.ServiceContainer
type mockServiceContainer struct {
	queueGroupClient *mockQueueGroupServiceClient
	serviceNotFound  bool
}

func (m *mockServiceContainer) BindModule(_ mono.Module) error {
	return nil
}

func (m *mockServiceContainer) SetEventBus(_ mono.EventBus) {}

func (m *mockServiceContainer) SetQueueGroupOptimisticWindow(_ time.Duration) {}

func (m *mockServiceContainer) SetMiddlewareChain(_ mono.MiddlewareChainRunner) {}

func (m *mockServiceContainer) RegisterChannelService(_ string, _ chan *mono.Msg, _ chan *mono.Msg) error {
	return nil
}

func (m *mockServiceContainer) RegisterRequestReplyService(_ string, _ mono.RequestReplyHandler) error {
	return nil
}

func (m *mockServiceContainer) RegisterQueueGroupService(_ string, _ ...mono.QGHP) error {
	return nil
}

func (m *mockServiceContainer) RegisterStreamConsumerService(_ string, _ mono.StreamConsumerConfig, _ mono.StreamConsumerHandler) error {
	return nil
}

func (m *mockServiceContainer) GetChannelService(_ string, _ string) (chan *mono.Msg, chan *mono.Msg, error) {
	return nil, nil, nil
}

func (m *mockServiceContainer) MustGetChannelService(_ string, _ string) (chan *mono.Msg, chan *mono.Msg) {
	return nil, nil
}

func (m *mockServiceContainer) GetRequestReplyService(_ string) (mono.RequestReplyServiceClient, error) {
	return nil, nil
}

func (m *mockServiceContainer) GetQueueGroupService(name string) (mono.QueueGroupServiceClient, error) {
	if m.serviceNotFound {
		return nil, errors.New("service not found")
	}
	if name == worker.ServiceName {
		return m.queueGroupClient, nil
	}
	return nil, errors.New("unknown service")
}

func (m *mockServiceContainer) GetStreamConsumerService(_ string) (mono.StreamConsumerServiceClient, error) {
	return nil, nil
}

func (m *mockServiceContainer) Has(name string) bool {
	return name == worker.ServiceName && !m.serviceNotFound
}

func (m *mockServiceContainer) Entries() []*mono.ServiceEntry {
	return nil
}

func (m *mockServiceContainer) Unregister(_ string) error {
	return nil
}

func (m *mockServiceContainer) StartChannelRouters(_ context.Context) {
}

func TestService_CreateJob_Success(t *testing.T) {
	tests := []struct {
		name    string
		req     *job.CreateJobRequest
		wantErr bool
	}{
		{
			name: "valid email job",
			req: &job.CreateJobRequest{
				Type: job.JobTypeEmail,
				Payload: map[string]any{
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
				Payload: map[string]any{
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
				Payload: map[string]any{
					"report_type": "sales",
					"format":      "pdf",
					"date_range": map[string]any{
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
			// Create fresh mocks for each test case
			jobStore := job.NewStore()
			mockClient := &mockQueueGroupServiceClient{sentData: make([][]byte, 0)}
			mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
			service := NewService(jobStore, mockContainer)

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

				if j.Status != job.JobStatusPending {
					t.Errorf("CreateJob() job status = %v, want %v", j.Status, job.JobStatusPending)
				}

				// Verify job was stored
				stored, err := jobStore.GetByID(j.ID)
				if err != nil {
					t.Errorf("Job not found in store: %v", err)
				}
				if stored.ID != j.ID {
					t.Errorf("Stored job ID = %v, want %v", stored.ID, j.ID)
				}

				// Verify job was sent to queue
				if len(mockClient.sentData) == 0 {
					t.Error("Job was not sent to queue")
				}
			}
		})
	}
}

func TestService_CreateJob_InvalidType(t *testing.T) {
	jobStore := job.NewStore()
	mockClient := &mockQueueGroupServiceClient{sentData: make([][]byte, 0)}
	mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
	service := NewService(jobStore, mockContainer)

	req := &job.CreateJobRequest{
		Type:     job.JobType("invalid_type"),
		Payload:  map[string]any{},
		Priority: 0,
	}

	_, err := service.CreateJob(context.Background(), req)
	if err != job.ErrInvalidJobType {
		t.Errorf("CreateJob() error = %v, want %v", err, job.ErrInvalidJobType)
	}
}

func TestService_CreateJob_InvalidPayload(t *testing.T) {
	jobStore := job.NewStore()
	mockClient := &mockQueueGroupServiceClient{sentData: make([][]byte, 0)}
	mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
	service := NewService(jobStore, mockContainer)

	tests := []struct {
		name    string
		jobType job.JobType
		payload map[string]any
	}{
		{
			name:    "email missing required fields",
			jobType: job.JobTypeEmail,
			payload: map[string]any{
				"to": "test@example.com",
				// missing subject and body
			},
		},
		{
			name:    "image processing missing required fields",
			jobType: job.JobTypeImageProcessing,
			payload: map[string]any{
				"image_url": "https://example.com/image.jpg",
				// missing output_path
			},
		},
		{
			name:    "report generation missing required fields",
			jobType: job.JobTypeReportGeneration,
			payload: map[string]any{
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
			if !errors.Is(err, job.ErrInvalidPayload) {
				t.Errorf("CreateJob() error = %v, want %v", err, job.ErrInvalidPayload)
			}
		})
	}
}

func TestService_CreateJob_QueueUnavailable(t *testing.T) {
	jobStore := job.NewStore()
	mockClient := &mockQueueGroupServiceClient{shouldFail: true, sentData: make([][]byte, 0)}
	mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
	service := NewService(jobStore, mockContainer)

	req := &job.CreateJobRequest{
		Type: job.JobTypeEmail,
		Payload: map[string]any{
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
	if len(jobs) > 0 && jobs[0].Status != job.JobStatusFailed {
		t.Errorf("Job status = %v, want %v after queue failure", jobs[0].Status, job.JobStatusFailed)
	}
}

func TestService_CreateJob_ServiceNotFound(t *testing.T) {
	jobStore := job.NewStore()
	mockContainer := &mockServiceContainer{serviceNotFound: true}
	service := NewService(jobStore, mockContainer)

	req := &job.CreateJobRequest{
		Type: job.JobTypeEmail,
		Payload: map[string]any{
			"to":      "test@example.com",
			"subject": "Test",
			"body":    "Test body",
		},
		Priority: 1,
	}

	_, err := service.CreateJob(context.Background(), req)
	if !errors.Is(err, job.ErrQueueUnavailable) {
		t.Errorf("CreateJob() error = %v, want %v", err, job.ErrQueueUnavailable)
	}
}

func TestService_GetJob(t *testing.T) {
	jobStore := job.NewStore()
	mockClient := &mockQueueGroupServiceClient{sentData: make([][]byte, 0)}
	mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
	service := NewService(jobStore, mockContainer)

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
	mockClient := &mockQueueGroupServiceClient{sentData: make([][]byte, 0)}
	mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
	service := NewService(jobStore, mockContainer)

	_, err := service.GetJob("non-existent-id")
	if err != job.ErrJobNotFound {
		t.Errorf("GetJob() error = %v, want %v", err, job.ErrJobNotFound)
	}
}

func TestService_ListJobs(t *testing.T) {
	jobStore := job.NewStore()
	mockClient := &mockQueueGroupServiceClient{sentData: make([][]byte, 0)}
	mockContainer := &mockServiceContainer{queueGroupClient: mockClient}
	service := NewService(jobStore, mockContainer)

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
