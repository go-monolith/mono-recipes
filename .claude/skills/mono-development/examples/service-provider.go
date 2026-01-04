// Example: Service Provider and Consumer
//
// This example demonstrates:
// - Registering RequestReply services
// - Declaring module dependencies
// - Consuming services from other modules
// - Type-safe service adapters

package serviceprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// ============================================================
// Payment Module (Service Provider)
// ============================================================

// PaymentRequest is the request payload
type PaymentRequest struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// PaymentResponse is the response payload
type PaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// PaymentModule provides payment processing services
type PaymentModule struct{}

// Compile-time interface checks
var (
	_ mono.Module                = (*PaymentModule)(nil)
	_ mono.ServiceProviderModule = (*PaymentModule)(nil)
)

func NewPaymentModule() *PaymentModule {
	return &PaymentModule{}
}

func (m *PaymentModule) Name() string { return "payment" }

func (m *PaymentModule) Start(_ context.Context) error {
	slog.Info("Payment module started")
	return nil
}

func (m *PaymentModule) Stop(_ context.Context) error {
	slog.Info("Payment module stopped")
	return nil
}

// RegisterServices registers the payment services
func (m *PaymentModule) RegisterServices(container mono.ServiceContainer) error {
	return container.RegisterRequestReplyService(
		"process-payment",
		m.handleProcessPayment,
	)
}

func (m *PaymentModule) handleProcessPayment(
	ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req PaymentRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	slog.Info("Processing payment",
		"orderID", req.OrderID,
		"amount", req.Amount,
		"currency", req.Currency)

	// Simulate payment processing
	response := PaymentResponse{
		TransactionID: fmt.Sprintf("TXN-%d", time.Now().UnixNano()),
		Status:        "approved",
		Message:       "Payment processed successfully",
	}

	return json.Marshal(response)
}

// ============================================================
// Order Module (Service Consumer)
// ============================================================

// OrderModule consumes payment services
type OrderModule struct {
	paymentContainer mono.ServiceContainer
}

// Compile-time interface checks
var (
	_ mono.Module                              = (*OrderModule)(nil)
	_ mono.DependentModule                     = (*OrderModule)(nil)
	_ mono.SetDependencyServiceContainerModule = (*OrderModule)(nil)
)

func NewOrderModule() *OrderModule {
	return &OrderModule{}
}

func (m *OrderModule) Name() string { return "order" }

func (m *OrderModule) Start(_ context.Context) error {
	slog.Info("Order module started")
	return nil
}

func (m *OrderModule) Stop(_ context.Context) error {
	slog.Info("Order module stopped")
	return nil
}

// Dependencies declares that order module depends on payment
func (m *OrderModule) Dependencies() []string {
	return []string{"payment"}
}

// SetDependencyServiceContainer receives the payment service container
func (m *OrderModule) SetDependencyServiceContainer(
	module string, container mono.ServiceContainer) {
	if module == "payment" {
		m.paymentContainer = container
	}
}

// ProcessOrder creates an order and processes payment
func (m *OrderModule) ProcessOrder(ctx context.Context, orderID string, amount float64) (*PaymentResponse, error) {
	var response PaymentResponse

	// Use helper for type-safe service call
	err := helper.CallRequestReplyService(
		ctx,
		m.paymentContainer,
		"process-payment",
		json.Marshal,
		json.Unmarshal,
		&PaymentRequest{
			OrderID:  orderID,
			Amount:   amount,
			Currency: "USD",
		},
		&response,
	)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	return &response, nil
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Service Provider Example ===")

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create modules
	paymentModule := NewPaymentModule()
	orderModule := NewOrderModule()

	// Register modules (framework resolves dependency order)
	if err := app.Register(paymentModule); err != nil {
		log.Fatalf("Failed to register payment module: %v", err)
	}
	if err := app.Register(orderModule); err != nil {
		log.Fatalf("Failed to register order module: %v", err)
	}

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Printf("Registered modules: %v\n", app.Modules())

	// Wait for services to be ready
	time.Sleep(100 * time.Millisecond)

	// Test the service communication
	fmt.Println("\n=== Testing Service Communication ===")

	response, err := orderModule.ProcessOrder(ctx, "ORD-001", 99.99)
	if err != nil {
		fmt.Printf("Order failed: %v\n", err)
	} else {
		fmt.Printf("Order processed: %s - %s\n", response.TransactionID, response.Status)
	}

	// Wait for shutdown signal
	fmt.Println("\nPress Ctrl+C to shutdown...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}

	fmt.Println("App stopped successfully")
}
