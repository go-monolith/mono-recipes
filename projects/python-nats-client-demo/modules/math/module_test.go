package math_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-monolith/mono/monotest"

	"github.com/example/python-nats-client-demo/modules/math"
	"github.com/example/python-nats-client-demo/modules/math/contract"
)

// TestMathModuleConformance is the module boundary check: valid name, a
// service registered with a valid subject, no boundary violation, clean
// Start/Stop. The exercise calls the provided contract the way the Python
// MathClient does (same subject, same JSON).
func TestMathModuleConformance(t *testing.T) {
	monotest.CheckModule(t, math.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			resp, err := contract.Calculate.Call(ctx, h.Container(),
				contract.CalculateRequest{Operation: contract.OpMultiply, A: 6, B: 7})
			if err != nil {
				return err
			}
			if resp.Result != 42 || resp.Error != "" {
				return fmt.Errorf("unexpected response: %+v", resp)
			}
			return nil
		}))
}

// TestCalculateErrorInResponse checks that a failed operation is reported in
// the response body, not as a service error, which the Python client expects.
func TestCalculateErrorInResponse(t *testing.T) {
	h := monotest.New(t, math.NewModule(), monotest.Strict())
	ctx := context.Background()
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	resp, err := contract.Calculate.Call(ctx, h.Container(),
		contract.CalculateRequest{Operation: contract.OpDivide, A: 1, B: 0})
	if err != nil {
		t.Fatalf("Calculate returned a service error: %v", err)
	}
	if resp.Error != "division by zero" {
		t.Errorf("Error = %q, want %q", resp.Error, "division by zero")
	}
}
