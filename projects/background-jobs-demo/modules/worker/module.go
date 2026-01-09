// Package worker provides job processing with QueueGroupService pattern.
package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/go-monolith/mono"
)

const (
	// ServiceName is the name of the QueueGroupService for job processing.
	ServiceName = "process-job"
	// QueueGroupEmail is the queue group for email jobs.
	QueueGroupEmail = "email-worker"
	// QueueGroupImageProcessing is the queue group for image processing jobs.
	QueueGroupImageProcessing = "image-processing-worker"
	// QueueGroupReportGeneration is the queue group for report generation jobs.
	QueueGroupReportGeneration = "report-generation-worker"
)

// Module provides job processing via QueueGroupService.
type Module struct {
	jobStore  *job.Store
	processor *Processor
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new worker module.
func NewModule(jobStore *job.Store) *Module {
	return &Module{
		jobStore: jobStore,
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "worker"
}

// RegisterServices registers the QueueGroupService with 3 queue groups for job processing.
// Each queue group handles a specific job type, filtering and ignoring non-matching jobs.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	m.processor = NewProcessor()

	return container.RegisterQueueGroupService(
		ServiceName,
		mono.QGHP{
			QueueGroup: QueueGroupEmail,
			Handler:    m.handleJobTypeEmail,
		},
		mono.QGHP{
			QueueGroup: QueueGroupImageProcessing,
			Handler:    m.handleJobTypeImageProcessing,
		},
		mono.QGHP{
			QueueGroup: QueueGroupReportGeneration,
			Handler:    m.handleJobTypeReportGeneration,
		},
	)
}

// Start starts the worker module.
func (m *Module) Start(_ context.Context) error {
	log.Println("[worker] Module started")
	return nil
}

// Stop stops the worker module.
func (m *Module) Stop(_ context.Context) error {
	log.Println("[worker] Module stopped")
	return nil
}

// handleJobTypeEmail handles email jobs, ignoring other job types.
func (m *Module) handleJobTypeEmail(ctx context.Context, msg *mono.Msg) error {
	return m.handleJobOfType(ctx, msg, job.JobTypeEmail, QueueGroupEmail)
}

// handleJobTypeImageProcessing handles image processing jobs, ignoring other job types.
func (m *Module) handleJobTypeImageProcessing(ctx context.Context, msg *mono.Msg) error {
	return m.handleJobOfType(ctx, msg, job.JobTypeImageProcessing, QueueGroupImageProcessing)
}

// handleJobTypeReportGeneration handles report generation jobs, ignoring other job types.
func (m *Module) handleJobTypeReportGeneration(ctx context.Context, msg *mono.Msg) error {
	return m.handleJobOfType(ctx, msg, job.JobTypeReportGeneration, QueueGroupReportGeneration)
}

// handleJobOfType processes jobs of the specified type, ignoring others.
func (m *Module) handleJobOfType(ctx context.Context, msg *mono.Msg, jobType job.JobType, workerID string) error {
	var j job.Job
	if err := json.Unmarshal(msg.Data, &j); err != nil {
		log.Printf("[%s] Error unmarshaling job: %v", workerID, err)
		return nil
	}

	if j.Type != jobType {
		return nil
	}

	return m.processJob(ctx, &j, workerID)
}

// processJob is the common job processing logic used by all type-specific handlers.
func (m *Module) processJob(ctx context.Context, j *job.Job, workerID string) error {
	log.Printf("[%s] Processing job %s (type=%s)", workerID, j.ID, j.Type)

	// Mark job as started
	if err := m.jobStore.SetStarted(j.ID, workerID); err != nil {
		log.Printf("[%s] Error setting job started: %v", workerID, err)
	}

	// Create progress callback
	progressFn := func(progress int, message string) {
		if err := m.jobStore.UpdateProgress(j.ID, progress, message); err != nil {
			log.Printf("[%s] Error updating progress: %v", workerID, err)
		}
	}

	// Process the job
	result, err := m.processor.Process(ctx, j, progressFn)
	if err != nil {
		log.Printf("[%s] Job %s context error: %v", workerID, j.ID, err)
		_ = m.jobStore.SetFailed(j.ID, err.Error())
		return nil // Don't retry on context errors
	}

	// Handle result
	if result.Success {
		_ = m.jobStore.SetCompleted(j.ID, result.Result)
		log.Printf("[%s] Job %s completed successfully", workerID, j.ID)
	} else {
		errMsg := "unknown error"
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		_ = m.jobStore.SetFailed(j.ID, errMsg)
		log.Printf("[%s] Job %s failed: %s", workerID, j.ID, errMsg)
	}

	return nil
}
