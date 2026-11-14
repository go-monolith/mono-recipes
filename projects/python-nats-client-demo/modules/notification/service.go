package notification

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/notification/contract"
)

// handleEmailSend handles email-send queue group messages. The contract has
// already decoded the JSON payload; a message that does not decode is logged
// by the framework and dropped (queue group messages are not redelivered).
// This is a fire-and-forget service - no response is sent back.
func (m *NotificationModule) handleEmailSend(_ context.Context, req contract.SendEmailRequest, _ *mono.Msg) error {
	// Validate required fields
	if req.To == "" || req.Subject == "" {
		log.Printf("[notification] Invalid email request: missing required fields")
		return nil
	}

	// Simulate email sending with random delay
	delay := time.Duration(100+rand.Intn(400)) * time.Millisecond
	time.Sleep(delay)

	// Simulate success/failure (90% success rate)
	if rand.Float32() < 0.9 {
		log.Printf("[notification] Email sent to %s: %q (took %v)", req.To, req.Subject, delay)
	} else {
		log.Printf("[notification] Failed to send email to %s: %q (simulated failure)", req.To, req.Subject)
	}

	return nil
}
