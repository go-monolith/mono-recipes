package payment

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-monolith/mono"
)

// paymentStore is an in-memory store for payment results.
type paymentStore struct {
	mu       sync.RWMutex
	payments map[string]*PaymentResult
}

func newPaymentStore() *paymentStore {
	return &paymentStore{
		payments: make(map[string]*PaymentResult),
	}
}

func (s *paymentStore) set(result *PaymentResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments[result.PaymentID] = result
}

func (s *paymentStore) get(paymentID string) *PaymentResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.payments[paymentID]
}

// getOrSetProcessing atomically checks if payment exists and sets to processing if not.
// Returns (existing result, true) if already exists, or (nil, false) if newly set.
func (s *paymentStore) getOrSetProcessing(paymentID string) (*PaymentResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.payments[paymentID]; existing != nil {
		return existing, true
	}
	s.payments[paymentID] = &PaymentResult{
		PaymentID: paymentID,
		Status:    PaymentStatusProcessing,
	}
	return nil, false
}

// handlePayments handles batch of payment messages from the stream.
func (m *PaymentModule) handlePayments(_ context.Context, msgs []*mono.Msg) error {
	for _, msg := range msgs {
		var req PaymentRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[payment] Error unmarshaling payment request: %v", err)
			msg.Term() // Don't retry malformed messages
			continue
		}

		// Atomically check for idempotency and mark as processing
		if existing, alreadyExists := m.store.getOrSetProcessing(req.PaymentID); alreadyExists {
			log.Printf("[payment] Payment %s already processed (status: %s)", req.PaymentID, existing.Status)
			msg.Ack()
			continue
		}

		// Simulate payment gateway call
		delay := time.Duration(200+rand.Intn(300)) * time.Millisecond
		time.Sleep(delay)

		// Simulate success/failure (95% success rate)
		var result *PaymentResult
		if rand.Float32() < 0.95 {
			result = &PaymentResult{
				PaymentID: req.PaymentID,
				Status:    PaymentStatusCompleted,
				Message:   "Payment processed successfully",
			}
			log.Printf("[payment] Payment %s completed for user %s ($%.2f) in %v",
				req.PaymentID, req.UserID, req.Amount, delay)
		} else {
			result = &PaymentResult{
				PaymentID: req.PaymentID,
				Status:    PaymentStatusFailed,
				Message:   "Payment gateway error (simulated)",
			}
			log.Printf("[payment] Payment %s failed for user %s ($%.2f)",
				req.PaymentID, req.UserID, req.Amount)
		}

		m.store.set(result)
		msg.Ack() // Only acknowledge after successful processing
	}

	return nil
}

// getStatus handles payment status queries via RequestReplyService.
func (m *PaymentModule) getStatus(_ context.Context, req StatusRequest, _ *mono.Msg) (StatusResponse, error) {
	result := m.store.get(req.PaymentID)
	if result == nil {
		return StatusResponse{
			PaymentID: req.PaymentID,
			Status:    PaymentStatusPending,
			Message:   "Payment not found or not yet processed",
		}, nil
	}

	return StatusResponse{
		PaymentID: result.PaymentID,
		Status:    result.Status,
		Message:   result.Message,
	}, nil
}
