package api

import (
	"context"

	"github.com/go-monolith/mono"

	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// workerAdapter is the driven adapter behind JobsPort: each method is one call
// of a worker contract through the container received in
// SetDependencyServiceContainer.
type workerAdapter struct {
	container mono.ServiceContainer
}

var _ JobsPort = workerAdapter{}

func (a workerAdapter) CreateJob(ctx context.Context, req workercontract.CreateJobRequest) (workercontract.CreateJobResponse, error) {
	return workercontract.CreateJob.Call(ctx, a.container, req)
}

func (a workerAdapter) GetJob(ctx context.Context, req workercontract.GetJobRequest) (workercontract.GetJobResponse, error) {
	return workercontract.GetJob.Call(ctx, a.container, req)
}

func (a workerAdapter) ListJobs(ctx context.Context, req workercontract.ListJobsRequest) (workercontract.ListJobsResponse, error) {
	return workercontract.ListJobs.Call(ctx, a.container, req)
}
