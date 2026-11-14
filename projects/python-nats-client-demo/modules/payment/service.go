package payment

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/payment/contract"
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
		Status:    contract.PaymentStatusProcessing,
	}
	return nil, false
}

// handlePayments handles a batch of payment messages from the stream.
// payments[i] is the decoded payload of msgs[i]; the handler acknowledges each
// message itself.
func (m *PaymentModule) handlePayments(_ context.Context, payments []contract.PaymentRequest, msgs []*mono.Msg) error {
	for i, req := range payments {
		msg := msgs[i]
		if req.PaymentID == "" {
			log.Printf("[payment] Invalid payment request: missing payment_id")
			_ = msg.Term() // Don't retry malformed messages
			continue
		}

		// Atomically check for idempotency and mark as processing
		if existing, alreadyExists := m.store.getOrSetProcessing(req.PaymentID); alreadyExists {
			log.Printf("[payment] Payment %s already processed (status: %s)", req.PaymentID, existing.Status)
			_ = msg.Ack()
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
				Status:    contract.PaymentStatusCompleted,
				Message:   "Payment processed successfully",
			}
			log.Printf("[payment] Payment %s completed for user %s ($%.2f) in %v",
				req.PaymentID, req.UserID, req.Amount, delay)
		} else {
			result = &PaymentResult{
				PaymentID: req.PaymentID,
				Status:    contract.PaymentStatusFailed,
				Message:   "Payment gateway error (simulated)",
			}
			log.Printf("[payment] Payment %s failed for user %s ($%.2f)",
				req.PaymentID, req.UserID, req.Amount)
		}

		m.store.set(result)
		_ = msg.Ack() // Only acknowledge after successful processing
	}

	return nil
}

// getStatus handles payment status queries via RequestReplyService.
func (m *PaymentModule) getStatus(_ context.Context, req contract.StatusRequest, _ *mono.Msg) (contract.StatusResponse, error) {
	result := m.store.get(req.PaymentID)
	if result == nil {
		return contract.StatusResponse{
			PaymentID: req.PaymentID,
			Status:    contract.PaymentStatusPending,
			Message:   "Payment not found or not yet processed",
		}, nil
	}

	return contract.StatusResponse{
		PaymentID: result.PaymentID,
		Status:    result.Status,
		Message:   result.Message,
	}, nil
}
