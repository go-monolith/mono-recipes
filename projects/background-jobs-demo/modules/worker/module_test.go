package worker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/background-jobs-demo/modules/worker"
	"github.com/example/background-jobs-demo/modules/worker/contract"
)

func raw(t testing.TB, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestWorkerModuleConformance checks the worker module against the Ports and
// Adapters rules. The exercise calls every contract it provides: create-job
// for one job of each type (which enqueues on process-job), get-job until each
// reached a terminal status in the queue group of its type, list-jobs, and
// process-job directly, as an external NATS client could.
func TestWorkerModuleConformance(t *testing.T) {
	payloads := map[string]json.RawMessage{
		"email":             raw(t, map[string]any{"to": "a@example.com", "subject": "Hi", "body": "Hello"}),
		"image_processing":  raw(t, map[string]any{"image_url": "https://example.com/a.jpg", "operations": []string{"resize"}, "output_path": "/out/a.jpg"}),
		"report_generation": raw(t, map[string]any{"report_type": "sales", "format": "pdf"}),
	}
	wantWorker := map[string]string{
		"email":             worker.QueueGroupEmail,
		"image_processing":  worker.QueueGroupImageProcessing,
		"report_generation": worker.QueueGroupReportGeneration,
	}

	monotest.CheckModule(t, worker.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			c := h.Container()
			ids := make(map[string]string)
			for jobType, payload := range payloads {
				resp, err := contract.CreateJob.Call(ctx, c, contract.CreateJobRequest{Type: jobType, Payload: payload, Priority: 1})
				if err != nil {
					return err
				}
				if resp.ErrorCode != "" || resp.Job == nil || resp.Job.Status != "pending" {
					return fmt.Errorf("create-job %s = %+v", jobType, resp)
				}
				ids[jobType] = resp.Job.ID
			}
			for jobType, id := range ids {
				j, err := waitTerminal(ctx, c, id)
				if err != nil {
					return fmt.Errorf("%s job: %w", jobType, err)
				}
				if j.WorkerID != wantWorker[jobType] {
					return fmt.Errorf("%s job processed by %q, want %q", jobType, j.WorkerID, wantWorker[jobType])
				}
				if j.Status == "completed" && !json.Valid(j.Result) {
					return fmt.Errorf("%s job completed without a valid result", jobType)
				}
			}
			list, err := contract.ListJobs.Call(ctx, c, contract.ListJobsRequest{Type: "email", Limit: 10})
			if err != nil {
				return err
			}
			if len(list.Jobs) != 1 || list.Jobs[0].ID != ids["email"] {
				return fmt.Errorf("list-jobs type=email = %+v", list.Jobs)
			}
			// An unknown job ID on process-job is accepted and ignored.
			return contract.ProcessJob.Send(ctx, c, contract.JobMessage{ID: "unknown", Type: "unknown"})
		}))
}

// TestCreateJobValidation checks the rejections of create-job: they are
// responses with a code, and nothing is recorded.
func TestCreateJobValidation(t *testing.T) {
	ctx := context.Background()
	h := monotest.New(t, worker.NewModule())
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	tests := []struct {
		name    string
		jobType string
		payload json.RawMessage
		want    string
	}{
		{"invalid type", "invalid_type", raw(t, map[string]any{}), contract.ErrorCodeInvalidJobType},
		{"email missing fields", "email", raw(t, map[string]any{"to": "a@example.com"}), contract.ErrorCodeInvalidPayload},
		{"image missing output_path", "image_processing", raw(t, map[string]any{"image_url": "https://example.com/a.jpg"}), contract.ErrorCodeInvalidPayload},
		{"report missing format", "report_generation", raw(t, map[string]any{"report_type": "sales"}), contract.ErrorCodeInvalidPayload},
		{"null payload", "email", nil, contract.ErrorCodeInvalidPayload},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := contract.CreateJob.Call(ctx, h.Container(), contract.CreateJobRequest{Type: tt.jobType, Payload: tt.payload})
			if err != nil {
				t.Fatal(err)
			}
			if resp.ErrorCode != tt.want || resp.Error == "" || resp.Job != nil {
				t.Errorf("create-job = %+v, want rejection %q", resp, tt.want)
			}
		})
	}

	list, err := contract.ListJobs.Call(ctx, h.Container(), contract.ListJobsRequest{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Jobs) != 0 {
		t.Errorf("rejected jobs were recorded: %+v", list.Jobs)
	}
	got, err := contract.GetJob.Call(ctx, h.Container(), contract.GetJobRequest{ID: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Found {
		t.Errorf("get-job missing = %+v, want not found", got)
	}
	if v := h.Report().Violations; len(v) != 0 {
		t.Errorf("boundary violations: %+v", v)
	}
}

// waitTerminal polls get-job until the job completed or failed (the processor
// fails a fraction of jobs at random on purpose).
func waitTerminal(ctx context.Context, c types.ServiceContainer, id string) (contract.Job, error) {
	for {
		resp, err := contract.GetJob.Call(ctx, c, contract.GetJobRequest{ID: id})
		if err != nil {
			return contract.Job{}, err
		}
		if !resp.Found || resp.Job == nil {
			return contract.Job{}, fmt.Errorf("job %s not found", id)
		}
		if resp.Job.Status == "completed" || resp.Job.Status == "failed" {
			return *resp.Job, nil
		}
		select {
		case <-ctx.Done():
			return contract.Job{}, fmt.Errorf("job %s still %s: %w", id, resp.Job.Status, ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}
