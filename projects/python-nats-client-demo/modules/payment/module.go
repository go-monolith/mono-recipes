package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

const (
	// StreamName is the JetStream stream name for payment requests.
	StreamName = "payment-requests"
	// ServiceName is the stream consumer service name.
	ServiceName = "payment-process"
)

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

// RegisterServices registers both the StreamConsumerService for payment processing
// and a RequestReplyService for status queries.
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
	// Register StreamConsumerService for durable payment processing
	// The stream subject must match what Python clients publish to
	streamSubject := "services.payment." + ServiceName
	config := mono.StreamConsumerConfig{
		Stream: mono.StreamConfig{
			Name:      StreamName,
			Subjects:  []string{streamSubject},
			Retention: mono.WorkQueuePolicy,
		},
		Fetch: mono.FetchConfig{
			BatchSize: 5,
		},
	}

	if err := container.RegisterStreamConsumerService(
		ServiceName,
		config,
		m.handlePayments,
	); err != nil {
		return fmt.Errorf("failed to register stream consumer service: %w", err)
	}

	// Register RequestReplyService for status queries
	if err := helper.RegisterTypedRequestReplyService(
		container, "status", json.Unmarshal, json.Marshal, m.getStatus,
	); err != nil {
		return fmt.Errorf("failed to register status service: %w", err)
	}

	log.Printf("[payment] Registered StreamConsumerService: services.payment.%s (stream: %s)", ServiceName, StreamName)
	log.Printf("[payment] Registered RequestReplyService: services.payment.status")
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
