// Package worker owns the background jobs: their state (an in-memory job
// store no other module can reach) and their processing with the
// QueueGroupService pattern. Other modules create and query jobs through
// worker/contract and never import this package.
package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/example/background-jobs-demo/modules/worker/contract"
)

// Queue groups of the process-job service. Every queue group receives each
// job message once and ignores the job types it does not handle, so each job
// type is processed by its own serial worker.
const (
	// QueueGroupEmail is the queue group for email jobs.
	QueueGroupEmail = "email-worker"
	// QueueGroupImageProcessing is the queue group for image processing jobs.
	QueueGroupImageProcessing = "image-processing-worker"
	// QueueGroupReportGeneration is the queue group for report generation jobs.
	QueueGroupReportGeneration = "report-generation-worker"
)

// Module provides job state and job processing via QueueGroupService.
type Module struct {
	jobStore  *job.Store
	processor *Processor
	container mono.ServiceContainer // own container, to enqueue on ProcessJob
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new worker module with an empty job store.
func NewModule() *Module {
	return &Module{
		jobStore: job.NewStore(),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "worker"
}

// RegisterServices registers the job state contracts (create-job, get-job,
// list-jobs) and the process-job contract with 3 queue groups. Each queue
// group handles a specific job type, ignoring non-matching jobs.
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	m.processor = NewProcessor()
	m.container = container

	if err := contract.CreateJob.Register(container, m.createJob); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.CreateJob.Name, err)
	}
	if err := contract.GetJob.Register(container, m.getJob); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetJob.Name, err)
	}
	if err := contract.ListJobs.Register(container, m.listJobs); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.ListJobs.Name, err)
	}

	return contract.ProcessJob.Register(container,
		types.TypedQGHP[contract.JobMessage]{
			QueueGroup: QueueGroupEmail,
			Handler:    m.handleJobTypeEmail,
		},
		types.TypedQGHP[contract.JobMessage]{
			QueueGroup: QueueGroupImageProcessing,
			Handler:    m.handleJobTypeImageProcessing,
		},
		types.TypedQGHP[contract.JobMessage]{
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
func (m *Module) handleJobTypeEmail(ctx context.Context, msg contract.JobMessage, _ *mono.Msg) error {
	return m.handleJobOfType(ctx, msg, job.JobTypeEmail, QueueGroupEmail)
}

// handleJobTypeImageProcessing handles image processing jobs, ignoring other job types.
func (m *Module) handleJobTypeImageProcessing(ctx context.Context, msg contract.JobMessage, _ *mono.Msg) error {
	return m.handleJobOfType(ctx, msg, job.JobTypeImageProcessing, QueueGroupImageProcessing)
}

// handleJobTypeReportGeneration handles report generation jobs, ignoring other job types.
func (m *Module) handleJobTypeReportGeneration(ctx context.Context, msg contract.JobMessage, _ *mono.Msg) error {
	return m.handleJobOfType(ctx, msg, job.JobTypeReportGeneration, QueueGroupReportGeneration)
}

// handleJobOfType processes jobs of the specified type, ignoring others.
func (m *Module) handleJobOfType(ctx context.Context, msg contract.JobMessage, jobType job.JobType, workerID string) error {
	if job.JobType(msg.Type) != jobType {
		return nil
	}

	j := &job.Job{
		ID:         msg.ID,
		Type:       jobType,
		Status:     job.JobStatusPending,
		Payload:    msg.Payload,
		Priority:   msg.Priority,
		MaxRetries: msg.MaxRetries,
		CreatedAt:  msg.CreatedAt,
	}
	return m.processJob(ctx, j, workerID)
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
