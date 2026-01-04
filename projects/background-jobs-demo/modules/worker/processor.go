// Package worker provides job processing with worker pool pattern.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
)

// Processor processes jobs based on their type.
type Processor struct {
	workerID string
}

// NewProcessor creates a new job processor.
func NewProcessor(workerID string) *Processor {
	return &Processor{
		workerID: workerID,
	}
}

// ProcessResult contains the result of processing a job.
type ProcessResult struct {
	Success bool
	Result  interface{}
	Error   error
}

// Process processes a job and returns the result.
// It also calls the progress callback during processing.
func (p *Processor) Process(ctx context.Context, j *job.Job, progressFn func(progress int, message string)) (*ProcessResult, error) {
	switch j.Type {
	case job.JobTypeEmail:
		return p.processEmail(ctx, j, progressFn)
	case job.JobTypeImageProcessing:
		return p.processImageProcessing(ctx, j, progressFn)
	case job.JobTypeReportGeneration:
		return p.processReportGeneration(ctx, j, progressFn)
	default:
		return nil, fmt.Errorf("unknown job type: %s", j.Type)
	}
}

// processEmail simulates sending an email.
func (p *Processor) processEmail(ctx context.Context, j *job.Job, progressFn func(progress int, message string)) (*ProcessResult, error) {
	var payload job.EmailPayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return &ProcessResult{Success: false, Error: err}, nil
	}

	log.Printf("[worker:%s] Processing email job %s: to=%s, subject=%s",
		p.workerID, j.ID, payload.To, payload.Subject)

	progressFn(10, "Validating email address")
	if err := p.sleepWithContext(ctx, 200*time.Millisecond); err != nil {
		return nil, err
	}

	// Simulate random failure (10% chance)
	if rand.Float32() < 0.1 {
		return &ProcessResult{
			Success: false,
			Error:   fmt.Errorf("SMTP connection failed"),
		}, nil
	}

	progressFn(30, "Connecting to SMTP server")
	if err := p.sleepWithContext(ctx, 300*time.Millisecond); err != nil {
		return nil, err
	}

	progressFn(60, "Sending email")
	if err := p.sleepWithContext(ctx, 500*time.Millisecond); err != nil {
		return nil, err
	}

	progressFn(90, "Confirming delivery")
	if err := p.sleepWithContext(ctx, 200*time.Millisecond); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"message_id": fmt.Sprintf("msg_%s", j.ID[:8]),
		"status":     "delivered",
		"recipient":  payload.To,
	}

	progressFn(100, "Email sent successfully")
	return &ProcessResult{Success: true, Result: result}, nil
}

// processImageProcessing simulates image processing.
func (p *Processor) processImageProcessing(ctx context.Context, j *job.Job, progressFn func(progress int, message string)) (*ProcessResult, error) {
	var payload job.ImageProcessingPayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return &ProcessResult{Success: false, Error: err}, nil
	}

	log.Printf("[worker:%s] Processing image job %s: url=%s, operations=%v",
		p.workerID, j.ID, payload.ImageURL, payload.Operations)

	progressFn(5, "Downloading image")
	if err := p.sleepWithContext(ctx, 500*time.Millisecond); err != nil {
		return nil, err
	}

	// Simulate random failure (15% chance)
	if rand.Float32() < 0.15 {
		return &ProcessResult{
			Success: false,
			Error:   fmt.Errorf("image download failed: connection timeout"),
		}, nil
	}

	progressFn(20, "Image downloaded")

	// Process each operation
	totalOps := len(payload.Operations)
	if totalOps == 0 {
		totalOps = 1
	}

	for i, op := range payload.Operations {
		progress := 20 + (60 * (i + 1) / totalOps)
		progressFn(progress, fmt.Sprintf("Applying operation: %s", op))
		if err := p.sleepWithContext(ctx, 400*time.Millisecond); err != nil {
			return nil, err
		}
	}

	progressFn(85, "Saving processed image")
	if err := p.sleepWithContext(ctx, 300*time.Millisecond); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"output_url":  payload.OutputPath,
		"operations":  payload.Operations,
		"size_before": "2.5MB",
		"size_after":  "1.2MB",
	}

	progressFn(100, "Image processing completed")
	return &ProcessResult{Success: true, Result: result}, nil
}

// processReportGeneration simulates report generation.
func (p *Processor) processReportGeneration(ctx context.Context, j *job.Job, progressFn func(progress int, message string)) (*ProcessResult, error) {
	var payload job.ReportGenerationPayload
	if err := json.Unmarshal(j.Payload, &payload); err != nil {
		return &ProcessResult{Success: false, Error: err}, nil
	}

	log.Printf("[worker:%s] Processing report job %s: type=%s, format=%s",
		p.workerID, j.ID, payload.ReportType, payload.Format)

	progressFn(5, "Initializing report generator")
	if err := p.sleepWithContext(ctx, 200*time.Millisecond); err != nil {
		return nil, err
	}

	// Simulate random failure (20% chance - reports are more complex)
	if rand.Float32() < 0.2 {
		return &ProcessResult{
			Success: false,
			Error:   fmt.Errorf("database query timeout"),
		}, nil
	}

	progressFn(15, "Querying database")
	if err := p.sleepWithContext(ctx, 800*time.Millisecond); err != nil {
		return nil, err
	}

	progressFn(35, "Processing data")
	if err := p.sleepWithContext(ctx, 600*time.Millisecond); err != nil {
		return nil, err
	}

	progressFn(55, "Generating charts")
	if err := p.sleepWithContext(ctx, 500*time.Millisecond); err != nil {
		return nil, err
	}

	progressFn(75, "Formatting output")
	if err := p.sleepWithContext(ctx, 400*time.Millisecond); err != nil {
		return nil, err
	}

	progressFn(90, "Saving report")
	if err := p.sleepWithContext(ctx, 300*time.Millisecond); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"report_type":  payload.ReportType,
		"format":       payload.Format,
		"download_url": fmt.Sprintf("/reports/%s.%s", j.ID, payload.Format),
		"pages":        rand.Intn(50) + 10,
		"size":         fmt.Sprintf("%.1fMB", float64(rand.Intn(10)+1)/2),
	}

	progressFn(100, "Report generation completed")
	return &ProcessResult{Success: true, Result: result}, nil
}

// sleepWithContext sleeps for the duration or until context is cancelled.
func (p *Processor) sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
