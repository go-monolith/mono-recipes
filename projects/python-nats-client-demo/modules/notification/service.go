package notification

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/go-monolith/mono"
)

// handleEmailSend handles email.send queue group messages.
// This is a fire-and-forget service - no response is sent back.
func (m *NotificationModule) handleEmailSend(_ context.Context, msg *mono.Msg) error {
	var req SendEmailRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("[notification] Error unmarshaling email request: %v", err)
		return nil // Don't retry malformed messages
	}

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
