package payment_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/payment"
	"github.com/example/python-nats-client-demo/modules/payment/contract"
)

// The payment module owns a JetStream stream consumer, which monotest does not
// emulate, so its boundary test runs the module in a real application with the
// embedded NATS server, JetStream in a temp dir and strict module boundaries.

// freePort returns a TCP port that is free right now, so the test does not
// collide with a running demo binary on the default 4222.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func startApp(t *testing.T) mono.MonoApplication {
	t.Helper()
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelError),
		mono.WithNATSPort(freePort(t)),
		mono.WithJetStreamStorageDir(t.TempDir()),
		// Default (non-strict) mode on purpose: violations are recorded in
		// app.ModuleBoundaries() instead of rejected, so the report check below
		// is meaningful. main.go runs strict once this report is clean.
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Register(payment.NewModule()); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	return app
}

// TestPaymentModuleInApp publishes to the stream through the contract (the
// same subject Python's js.publish uses), including a malformed message in the
// same batch, then polls the status contract until the valid payment is
// processed, and checks the module's boundary report.
func TestPaymentModuleInApp(t *testing.T) {
	app := startApp(t)
	ctx := context.Background()
	services := app.Services("payment")

	// A malformed message first: it must be terminated, not block the batch.
	client, err := services.GetStreamConsumerService(contract.ProcessPayment.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Publish(ctx, []byte("not json")); err != nil {
		t.Fatalf("publish malformed: %v", err)
	}

	req := contract.PaymentRequest{PaymentID: "pay-test-1", UserID: "user-1", SubscriptionID: "sub-monthly", Amount: 9.99}
	if _, err := contract.ProcessPayment.Publish(ctx, services, req); err != nil {
		t.Fatalf("ProcessPayment.Publish: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := contract.Status.Call(ctx, services, contract.StatusRequest{PaymentID: req.PaymentID})
		if err != nil {
			t.Fatalf("Status.Call: %v", err)
		}
		if resp.Status == contract.PaymentStatusCompleted || resp.Status == contract.PaymentStatusFailed {
			break // the module simulates a 5% gateway failure
		}
		if time.Now().After(deadline) {
			t.Fatalf("payment not processed in time, last status %q", resp.Status)
		}
		time.Sleep(50 * time.Millisecond)
	}

	for _, r := range app.ModuleBoundaries() {
		if len(r.Violations) != 0 {
			t.Errorf("module %s: boundary violations: %+v", r.Module, r.Violations)
		}
	}
}

// TestStatusUnknownPayment checks the default answer for an unknown payment.
func TestStatusUnknownPayment(t *testing.T) {
	app := startApp(t)
	resp, err := contract.Status.Call(context.Background(), app.Services("payment"), contract.StatusRequest{PaymentID: "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.PaymentStatusPending {
		t.Errorf("Status = %q, want %q", resp.Status, contract.PaymentStatusPending)
	}
}
