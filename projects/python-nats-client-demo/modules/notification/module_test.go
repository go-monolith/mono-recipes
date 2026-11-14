package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"

	"github.com/example/python-nats-client-demo/modules/notification"
	"github.com/example/python-nats-client-demo/modules/notification/contract"
)

// TestNotificationModuleConformance is the module boundary check. The
// exercise sends through the provided contract (same subject and JSON the
// Python EmailClient publishes) and waits for the worker to handle it.
func TestNotificationModuleConformance(t *testing.T) {
	monotest.CheckModule(t, notification.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return contract.SendEmail.Send(ctx, h.Container(), contract.SendEmailRequest{
				To: "user@example.com", Subject: "Welcome", Body: "Thanks for signing up.",
			})
		}))
}

// TestEmailSendHandled checks that a sent email reaches the queue group worker.
func TestEmailSendHandled(t *testing.T) {
	h := monotest.New(t, notification.NewModule(), monotest.Strict())
	ctx := context.Background()
	if err := h.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = h.Stop(ctx) }()

	for _, req := range []contract.SendEmailRequest{
		{To: "a@example.com", Subject: "One", Body: "1"},
		{To: "", Subject: "missing recipient is ignored", Body: "2"},
	} {
		if err := contract.SendEmail.Send(ctx, h.Container(), req); err != nil {
			t.Fatalf("Send(%+v): %v", req, err)
		}
	}

	name := contract.SendEmail.Name
	deadline := time.Now().Add(5 * time.Second)
	for h.Container().Calls(name) < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("handled %d of 2 emails", h.Container().Calls(name))
		}
		time.Sleep(10 * time.Millisecond)
	}
}
