package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/go-monolith/mono/monotest"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/rate-limiting-middleware/modules/api"
	"github.com/example/rate-limiting-middleware/modules/api/contract"
)

// nopLogger discards module logs.
type nopLogger struct{}

var _ types.Logger = nopLogger{}

func (nopLogger) Debug(string, ...any)             {}
func (nopLogger) Info(string, ...any)              {}
func (nopLogger) Warn(string, ...any)              {}
func (nopLogger) Error(string, ...any)             {}
func (l nopLogger) With(...any) types.Logger       { return l }
func (l nopLogger) WithModule(string) types.Logger { return l }
func (l nopLogger) WithError(error) types.Logger   { return l }

// TestAPIModuleConformance is the module boundary check. The exercise calls
// every provided contract. The rate limiting middleware is not part of it:
// monotest has no middleware chain, and the middleware has its own tests.
func TestAPIModuleConformance(t *testing.T) {
	monotest.CheckModule(t, api.NewModule(nopLogger{}),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			if _, err := contract.GetData.Call(ctx, h.Container(), contract.GetDataRequest{}); err != nil {
				return err
			}
			if _, err := contract.CreateOrder.Call(ctx, h.Container(),
				contract.OrderRequest{ProductID: "prod-123", Quantity: 2, Price: 29.99}); err != nil {
				return err
			}
			_, err := contract.GetStatus.Call(ctx, h.Container(), contract.GetStatusRequest{})
			return err
		}))
}

func startHarness(t *testing.T) *monotest.Harness {
	t.Helper()
	h := monotest.New(t, api.NewModule(nopLogger{}), monotest.Strict())
	if err := h.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Stop(context.Background()) })
	return h
}

// TestCreateOrderThroughContract checks the response and that a validation
// error reaches the caller as a service error.
func TestCreateOrderThroughContract(t *testing.T) {
	h := startHarness(t)
	ctx := context.Background()

	order, err := contract.CreateOrder.Call(ctx, h.Container(),
		contract.OrderRequest{ProductID: "prod-123", Quantity: 2, Price: 10})
	if err != nil {
		t.Fatal(err)
	}
	if order.Total != 20 || order.Status != "pending" || order.OrderID == "" {
		t.Errorf("unexpected order: %+v", order)
	}

	_, err = contract.CreateOrder.Call(ctx, h.Container(), contract.OrderRequest{Quantity: 1, Price: 1})
	if err == nil || !strings.Contains(err.Error(), "product_id is required") {
		t.Errorf("err = %v, want product_id is required", err)
	}
}

// TestRequestBodies checks what the services accept on the wire: an empty body
// for the parameterless services, and JSON only for create-order.
func TestRequestBodies(t *testing.T) {
	h := startHarness(t)
	bus := h.EventBus()
	ctx := context.Background()

	for _, subject := range []string{contract.GetData.Subject(), contract.GetStatus.Subject()} {
		for _, body := range []string{"", "{}"} {
			msg, err := bus.RequestWithContext(ctx, subject, []byte(body))
			if err != nil {
				t.Fatalf("%s %q: %v", subject, body, err)
			}
			if !strings.HasPrefix(string(msg.Data), "{") {
				t.Errorf("%s %q: reply %q is not a JSON object", subject, body, msg.Data)
			}
		}
	}

	msg, err := bus.RequestWithContext(ctx, contract.CreateOrder.Subject(), []byte("invalid json"))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Header.Get(types.HeaderError) != "true" ||
		!strings.Contains(msg.Header.Get(types.HeaderErrorMessage), "failed to unmarshal request") {
		t.Errorf("invalid JSON to create-order: got header %v, data %q; want an unmarshal error reply", msg.Header, msg.Data)
	}
}
