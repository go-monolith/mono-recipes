package api

import (
	"context"

	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// JobsPort is the driven port the job service needs from the worker module,
// which owns job state: create (record + enqueue) a job and query jobs. The
// api module owns it; it speaks the worker contract's DTOs and says nothing
// about NATS.
type JobsPort interface {
	CreateJob(ctx context.Context, req workercontract.CreateJobRequest) (workercontract.CreateJobResponse, error)
	GetJob(ctx context.Context, req workercontract.GetJobRequest) (workercontract.GetJobResponse, error)
	ListJobs(ctx context.Context, req workercontract.ListJobsRequest) (workercontract.ListJobsResponse, error)
}
