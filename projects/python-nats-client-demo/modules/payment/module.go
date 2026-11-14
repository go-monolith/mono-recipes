package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono"

	"github.com/example/python-nats-client-demo/modules/payment/contract"
)

// StreamName is the JetStream stream name for payment requests.
const StreamName = "payment-requests"

// PaymentModule provides payment processing via StreamConsumerService
// and status queries via RequestReplyService.
type PaymentModule struct {
	store *paymentStore
}

// Compile-time interface checks.
var (
	_ mono.Module                = (*PaymentModule)(nil)
	_ mono.ServiceProviderModule = (*PaymentModule)(nil)
)

// NewModule creates a new PaymentModule.
func NewModule() *PaymentModule {
	return &PaymentModule{
		store: newPaymentStore(),
	}
}

// Name returns the module name.
func (m *PaymentModule) Name() string {
	return "payment"
}

// RegisterServices binds both handlers to the module's own contracts: the
// StreamConsumerService for durable payment processing and the
// RequestReplyService for status queries.
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
	// The stream subject comes from the contract, so it is exactly the
	// subject Python clients publish to (services.payment.payment-process).
	config := mono.StreamConsumerConfig{
		Stream: mono.StreamConfig{
			Name:      StreamName,
			Subjects:  []string{contract.ProcessPayment.Subject()},
			Retention: mono.WorkQueuePolicy,
		},
		Fetch: mono.FetchConfig{
			BatchSize: 5,
			// A fetch returns when the batch is full or after Timeout
			// (default 5s). Keep it short so a few payments are processed
			// within the 2s demo.py waits before querying their status.
			Timeout: time.Second,
		},
	}

	if err := contract.ProcessPayment.WithUnmarshaler(decodePaymentOrZero).
		Register(container, config, m.handlePayments); err != nil {
		return fmt.Errorf("failed to register stream consumer service: %w", err)
	}

	if err := contract.Status.Register(container, m.getStatus); err != nil {
		return fmt.Errorf("failed to register status service: %w", err)
	}

	log.Printf("[payment] Registered StreamConsumerService: %s (stream: %s)", contract.ProcessPayment.Subject(), StreamName)
	log.Printf("[payment] Registered RequestReplyService: %s", contract.Status.Subject())
	return nil
}

// decodePaymentOrZero is the provider-side decoder of the payment stream.
//
// The typed stream consumer fails the whole batch when one message does not
// decode, leaving every message of the batch unacknowledged until redelivery.
// A malformed message published by a client would then hold up the valid
// ones. Instead, a message that does not decode becomes a zero
// PaymentRequest, which handlePayments terminates (no payment_id).
func decodePaymentOrZero(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		log.Printf("[payment] Malformed payment request: %v", err)
		if req, ok := v.(*contract.PaymentRequest); ok {
			*req = contract.PaymentRequest{}
		}
	}
	return nil
}

// Start initializes the payment module.
func (m *PaymentModule) Start(_ context.Context) error {
	log.Println("[payment] Module started successfully")
	return nil
}

// Stop gracefully stops the payment module.
func (m *PaymentModule) Stop(_ context.Context) error {
	log.Println("[payment] Module stopped")
	return nil
}
