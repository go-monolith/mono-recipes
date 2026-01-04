package worker

import (
	"context"
	"testing"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
)

func TestProcessor_Process_Email(t *testing.T) {
	processor := NewProcessor("test-worker")

	payload := &job.EmailPayload{
		To:      "test@example.com",
		Subject: "Test",
		Body:    "Test body",
	}

	j := job.NewJobWithPayload(job.JobTypeEmail, payload, 1)

	progressCalled := false
	progressFn := func(progress int, message string) {
		progressCalled = true
		if progress < 0 || progress > 100 {
			t.Errorf("Invalid progress value: %d", progress)
		}
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, j, progressFn)

	if err != nil {
		t.Errorf("Process() error = %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}

	if !progressCalled {
		t.Error("Progress callback was not called")
	}
}

func TestProcessor_Process_ImageProcessing(t *testing.T) {
	processor := NewProcessor("test-worker")

	payload := &job.ImageProcessingPayload{
		ImageURL:   "https://example.com/image.jpg",
		Operations: []string{"resize", "watermark"},
		OutputPath: "/output/image.jpg",
	}

	j := job.NewJobWithPayload(job.JobTypeImageProcessing, payload, 1)

	progressCalled := false
	progressFn := func(progress int, message string) {
		progressCalled = true
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, j, progressFn)

	if err != nil {
		t.Errorf("Process() error = %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}

	if !progressCalled {
		t.Error("Progress callback was not called")
	}
}

func TestProcessor_Process_ReportGeneration(t *testing.T) {
	processor := NewProcessor("test-worker")

	payload := &job.ReportGenerationPayload{
		ReportType: "sales",
		Format:     "pdf",
		StartDate:  "2024-01-01",
		EndDate:    "2024-01-31",
	}

	j := job.NewJobWithPayload(job.JobTypeReportGeneration, payload, 1)

	progressCalled := false
	progressFn := func(progress int, message string) {
		progressCalled = true
	}

	ctx := context.Background()
	result, err := processor.Process(ctx, j, progressFn)

	if err != nil {
		t.Errorf("Process() error = %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}

	if !progressCalled {
		t.Error("Progress callback was not called")
	}
}

func TestProcessor_Process_InvalidType(t *testing.T) {
	processor := NewProcessor("test-worker")

	j := job.NewJob(job.JobType("invalid_type"), []byte("{}"), 1)

	progressFn := func(progress int, message string) {}

	ctx := context.Background()
	result, err := processor.Process(ctx, j, progressFn)

	if err == nil {
		t.Error("Process() expected error for invalid job type")
	}

	if result != nil && result.Success {
		t.Error("Process() should not succeed for invalid job type")
	}
}

func TestProcessor_Process_ContextCancellation(t *testing.T) {
	processor := NewProcessor("test-worker")

	payload := &job.EmailPayload{
		To:      "test@example.com",
		Subject: "Test",
		Body:    "Test body",
	}

	j := job.NewJobWithPayload(job.JobTypeEmail, payload, 1)

	progressFn := func(progress int, message string) {}

	// Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := processor.Process(ctx, j, progressFn)

	if err != context.Canceled {
		t.Errorf("Process() error = %v, want %v", err, context.Canceled)
	}

	if result != nil {
		t.Error("Process() should return nil result on context cancellation")
	}
}

func TestProcessor_Process_Timeout(t *testing.T) {
	processor := NewProcessor("test-worker")

	payload := &job.ImageProcessingPayload{
		ImageURL:   "https://example.com/image.jpg",
		Operations: []string{"resize"},
		OutputPath: "/output/image.jpg",
	}

	j := job.NewJobWithPayload(job.JobTypeImageProcessing, payload, 1)

	progressFn := func(progress int, message string) {}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	result, err := processor.Process(ctx, j, progressFn)

	if err != context.DeadlineExceeded {
		t.Errorf("Process() error = %v, want %v", err, context.DeadlineExceeded)
	}

	if result != nil {
		t.Error("Process() should return nil result on timeout")
	}
}

func TestCalculateRetryDelay(t *testing.T) {
	config := DefaultPoolConfig()

	tests := []struct {
		name       string
		retryCount int
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "first retry",
			retryCount: 1,
			wantMin:    1 * time.Second,
			wantMax:    1 * time.Second,
		},
		{
			name:       "second retry",
			retryCount: 2,
			wantMin:    2 * time.Second,
			wantMax:    2 * time.Second,
		},
		{
			name:       "third retry",
			retryCount: 3,
			wantMin:    4 * time.Second,
			wantMax:    4 * time.Second,
		},
		{
			name:       "fourth retry",
			retryCount: 4,
			wantMin:    8 * time.Second,
			wantMax:    8 * time.Second,
		},
		{
			name:       "fifth retry (capped at max)",
			retryCount: 5,
			wantMin:    16 * time.Second,
			wantMax:    16 * time.Second,
		},
		{
			name:       "tenth retry (capped at max)",
			retryCount: 10,
			wantMin:    time.Minute,
			wantMax:    time.Minute,
		},
	}

	worker := NewWorker("test", config, nil, nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := worker.calculateRetryDelay(tt.retryCount)

			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("calculateRetryDelay(%d) = %v, want between %v and %v",
					tt.retryCount, delay, tt.wantMin, tt.wantMax)
			}

			// Ensure delay never exceeds max
			if delay > config.MaxRetryDelay {
				t.Errorf("calculateRetryDelay(%d) = %v, exceeds max %v",
					tt.retryCount, delay, config.MaxRetryDelay)
			}
		})
	}
}
