package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/background-jobs-demo/modules/api"
	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// standInWorker is a NATS-free stand-in for the worker module that knows one
// job, registered through the worker's own contracts. create-job rejects the
// "unknown" job type the way the worker does.
func standInWorker(t *testing.T, bus *monotest.EventBus) *monotest.Container {
	t.Helper()
	c := monotest.NewContainer("worker", bus)
	now := time.Now()
	stored := workercontract.Job{ID: "job-1", Type: "email", Status: "pending", Payload: json.RawMessage(`{"to":"a@example.com"}`), MaxRetries: 3, CreatedAt: now, UpdatedAt: now}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(workercontract.CreateJob.Register(c, func(_ context.Context, req workercontract.CreateJobRequest, _ *types.Msg) (workercontract.CreateJobResponse, error) {
		if req.Type == "unknown" {
			return workercontract.CreateJobResponse{ErrorCode: workercontract.ErrorCodeInvalidJobType, Error: "invalid job type"}, nil
		}
		return workercontract.CreateJobResponse{Job: &stored}, nil
	}))
	must(workercontract.GetJob.Register(c, func(_ context.Context, req workercontract.GetJobRequest, _ *types.Msg) (workercontract.GetJobResponse, error) {
		if req.ID != stored.ID {
			return workercontract.GetJobResponse{Found: false}, nil
		}
		return workercontract.GetJobResponse{Job: &stored, Found: true}, nil
	}))
	must(workercontract.ListJobs.Register(c, func(context.Context, workercontract.ListJobsRequest, *types.Msg) (workercontract.ListJobsResponse, error) {
		return workercontract.ListJobsResponse{Jobs: []workercontract.Job{stored}}, nil
	}))
	return c
}

// TestAPIModuleConformance checks the api module against the Ports and
// Adapters rules. The exercise hits every route, so every consumed worker
// contract is called through the adapter, and checks the JSON shapes and
// status codes of the HTTP API.
func TestAPIModuleConformance(t *testing.T) {
	bus := monotest.NewEventBus()
	module := api.NewModule(0)
	monotest.CheckModule(t, module,
		monotest.WithDependency("worker", standInWorker(t, bus)),
		monotest.WithExercise(func(context.Context, *monotest.Harness) error {
			body := map[string]any{
				"type":    "email",
				"payload": map[string]any{"to": "a@example.com", "subject": "Hi", "body": "Hello"},
			}
			var created api.JobResponse
			if err := do(module, http.MethodPost, "/api/v1/jobs", body, http.StatusCreated, &created); err != nil {
				return err
			}
			if created.ID != "job-1" || created.Status != "pending" || created.MaxRetries != 3 {
				return fmt.Errorf("created = %+v", created)
			}
			var got api.JobResponse
			if err := do(module, http.MethodGet, "/api/v1/jobs/job-1", nil, http.StatusOK, &got); err != nil {
				return err
			}
			if got.ID != "job-1" || got.CreatedAt == "" {
				return fmt.Errorf("get = %+v", got)
			}
			if err := do(module, http.MethodGet, "/api/v1/jobs/missing", nil, http.StatusNotFound, nil); err != nil {
				return err
			}
			var list struct {
				Jobs  []api.JobResponse `json:"jobs"`
				Count int               `json:"count"`
				Limit int               `json:"limit"`
			}
			if err := do(module, http.MethodGet, "/api/v1/jobs?type=email", nil, http.StatusOK, &list); err != nil {
				return err
			}
			if list.Count != 1 || len(list.Jobs) != 1 || list.Limit != 50 {
				return fmt.Errorf("list = %+v", list)
			}
			// Worker rejections map to 400.
			bad := map[string]any{"type": "unknown", "payload": map[string]any{}}
			return do(module, http.MethodPost, "/api/v1/jobs", bad, http.StatusBadRequest, nil)
		}))
}

// do sends an HTTP request to the module's Fiber app and checks the status.
func do(module *api.Module, method, path string, body any, wantStatus int, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := module.GetApp().Test(req, 5000)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		return fmt.Errorf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, wantStatus, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
