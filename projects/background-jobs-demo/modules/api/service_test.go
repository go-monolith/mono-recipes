package api

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// fakeWorker is the state behind the stand-in worker container: the jobs it
// "created", and the rejection code create-job answers with, if any.
type fakeWorker struct {
	mu     sync.Mutex
	jobs   []workercontract.Job
	reject string
}

// standInWorker returns a NATS-free stand-in for the worker module, its
// services registered through the worker's own contracts and dispatched on
// the bus.
func standInWorker(t *testing.T) (*monotest.Container, *fakeWorker) {
	t.Helper()
	bus := monotest.NewEventBus()
	t.Cleanup(bus.Close)
	c := monotest.NewContainer("worker", bus)
	fake := &fakeWorker{}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(workercontract.CreateJob.Register(c, func(_ context.Context, req workercontract.CreateJobRequest, _ *types.Msg) (workercontract.CreateJobResponse, error) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		if fake.reject != "" {
			return workercontract.CreateJobResponse{ErrorCode: fake.reject, Error: "rejected: " + fake.reject}, nil
		}
		now := time.Now()
		j := workercontract.Job{ID: "job-" + req.Type, Type: req.Type, Status: "pending", Payload: req.Payload, Priority: req.Priority, CreatedAt: now, UpdatedAt: now}
		fake.jobs = append(fake.jobs, j)
		return workercontract.CreateJobResponse{Job: &j}, nil
	}))
	must(workercontract.GetJob.Register(c, func(_ context.Context, req workercontract.GetJobRequest, _ *types.Msg) (workercontract.GetJobResponse, error) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		for _, j := range fake.jobs {
			if j.ID == req.ID {
				return workercontract.GetJobResponse{Job: &j, Found: true}, nil
			}
		}
		return workercontract.GetJobResponse{Found: false}, nil
	}))
	must(workercontract.ListJobs.Register(c, func(_ context.Context, req workercontract.ListJobsRequest, _ *types.Msg) (workercontract.ListJobsResponse, error) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		out := []workercontract.Job{}
		for _, j := range fake.jobs {
			if req.Type == "" || j.Type == req.Type {
				out = append(out, j)
			}
		}
		if req.Offset >= len(out) {
			return workercontract.ListJobsResponse{Jobs: []workercontract.Job{}}, nil
		}
		end := min(req.Offset+req.Limit, len(out))
		return workercontract.ListJobsResponse{Jobs: out[req.Offset:end]}, nil
	}))
	must(c.Dispatch(context.Background()))
	return c, fake
}

// newTestService wires a Service to the container through the real adapter.
func newTestService(c *monotest.Container) *Service {
	return NewService(workerAdapter{container: c})
}

func TestService_CreateJob_Success(t *testing.T) {
	worker, fake := standInWorker(t)
	service := newTestService(worker)

	payload := map[string]any{"to": "test@example.com", "subject": "Test", "body": "Test body"}
	j, err := service.CreateJob(context.Background(), "email", payload, 1)
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if j.Type != "email" || j.Status != "pending" || j.Priority != 1 {
		t.Errorf("CreateJob() = %+v", j)
	}
	if len(fake.jobs) != 1 || string(fake.jobs[0].Payload) != `{"body":"Test body","subject":"Test","to":"test@example.com"}` {
		t.Errorf("worker received %+v", fake.jobs)
	}
}

func TestService_CreateJob_Rejections(t *testing.T) {
	tests := []struct {
		code string
		want error
	}{
		{workercontract.ErrorCodeInvalidJobType, ErrInvalidJobType},
		{workercontract.ErrorCodeInvalidPayload, ErrInvalidPayload},
		{workercontract.ErrorCodeQueueUnavailable, ErrQueueUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			worker, fake := standInWorker(t)
			fake.reject = tt.code
			_, err := newTestService(worker).CreateJob(context.Background(), "email", map[string]any{}, 0)
			if !errors.Is(err, tt.want) {
				t.Errorf("CreateJob() error = %v, want %v", err, tt.want)
			}
			// The worker's reason is passed through verbatim (shown in the HTTP body).
			if err != nil && err.Error() != "rejected: "+tt.code {
				t.Errorf("CreateJob() error message = %q", err.Error())
			}
		})
	}
}

func TestService_CreateJob_WorkerUnreachable(t *testing.T) {
	// The worker never registered create-job.
	service := newTestService(monotest.NewContainer("worker", monotest.NewEventBus()))
	_, err := service.CreateJob(context.Background(), "email", map[string]any{}, 0)
	if !errors.Is(err, ErrQueueUnavailable) {
		t.Errorf("CreateJob() error = %v, want %v", err, ErrQueueUnavailable)
	}
}

func TestService_GetJob(t *testing.T) {
	worker, _ := standInWorker(t)
	service := newTestService(worker)
	ctx := context.Background()

	created, err := service.CreateJob(ctx, "email", map[string]any{"to": "a@example.com"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.GetJob(ctx, created.ID)
	if err != nil || got.ID != created.ID {
		t.Errorf("GetJob() = %+v, %v", got, err)
	}
	if _, err := service.GetJob(ctx, "non-existent-id"); !errors.Is(err, ErrJobNotFound) {
		t.Errorf("GetJob(missing) error = %v, want %v", err, ErrJobNotFound)
	}
}

func TestService_ListJobs(t *testing.T) {
	worker, _ := standInWorker(t)
	service := newTestService(worker)
	ctx := context.Background()

	for _, jobType := range []string{"email", "image_processing", "report_generation"} {
		if _, err := service.CreateJob(ctx, jobType, map[string]any{}, 0); err != nil {
			t.Fatal(err)
		}
	}
	all, err := service.ListJobs(ctx, "", "", 10, 0)
	if err != nil || len(all) != 3 {
		t.Errorf("ListJobs() = %d jobs, %v; want 3", len(all), err)
	}
	page, _ := service.ListJobs(ctx, "", "", 2, 2)
	if len(page) != 1 {
		t.Errorf("ListJobs(limit 2, offset 2) = %d jobs, want 1", len(page))
	}
	emails, _ := service.ListJobs(ctx, "", "email", 10, 0)
	if len(emails) != 1 {
		t.Errorf("ListJobs(type email) = %d jobs, want 1", len(emails))
	}
}
