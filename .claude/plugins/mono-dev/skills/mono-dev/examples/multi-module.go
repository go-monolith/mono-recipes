// Example: Multi-Module Communication
//
// This example demonstrates:
// - Multiple modules with dependencies
// - RequestReply service registration and consumption
// - Service adapter pattern for type-safe calls
// - Dependency resolution and startup order
// - Module coordination for order processing flow

package multimodule

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
// Shared Types
// ============================================================

// InventoryRequest for checking stock
type InventoryRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// InventoryResponse from stock check
type InventoryResponse struct {
	Available bool   `json:"available"`
	Stock     int    `json:"stock"`
	Message   string `json:"message"`
}

// PaymentRequest for processing payment
type PaymentRequest struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// PaymentResponse from payment processing
type PaymentResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// CreateOrderRequest for order creation
type CreateOrderRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

// CreateOrderResponse from order creation
type CreateOrderResponse struct {
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id,omitempty"`
	Message       string `json:"message"`
}

// ============================================================
// Inventory Module (Service Provider - No Dependencies)
// ============================================================

type InventoryModule struct {
	stock map[string]int
}

var (
	_ mono.Module                = (*InventoryModule)(nil)
	_ mono.ServiceProviderModule = (*InventoryModule)(nil)
)

func NewInventoryModule() *InventoryModule {
	return &InventoryModule{
		stock: map[string]int{
			"laptop":   10,
			"mouse":    50,
			"keyboard": 25,
		},
	}
}

func (m *InventoryModule) Name() string { return "inventory" }

func (m *InventoryModule) Start(_ context.Context) error {
	slog.Info("Inventory module started", "products", len(m.stock))
	return nil
}

func (m *InventoryModule) Stop(_ context.Context) error {
	slog.Info("Inventory module stopped")
	return nil
}

func (m *InventoryModule) RegisterServices(container mono.ServiceContainer) error {
	return container.RegisterRequestReplyService(
		"check-stock",
		m.handleCheckStock,
	)
}

func (m *InventoryModule) handleCheckStock(
	ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req InventoryRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	stock, exists := m.stock[req.ProductID]
	if !exists {
		return json.Marshal(InventoryResponse{
			Available: false,
			Stock:     0,
			Message:   "Product not found",
		})
	}

	available := stock >= req.Quantity
	slog.Info("Stock check",
		"productID", req.ProductID,
		"requested", req.Quantity,
		"available", stock,
		"canFulfill", available)

	return json.Marshal(InventoryResponse{
		Available: available,
		Stock:     stock,
		Message:   fmt.Sprintf("Stock: %d units", stock),
	})
}

// ============================================================
// Payment Module (Service Provider - No Dependencies)
// ============================================================

type PaymentModule struct {
	txnCounter int
}

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

	m.txnCounter++
	txnID := fmt.Sprintf("TXN-%06d", m.txnCounter)

	slog.Info("Processing payment",
		"orderID", req.OrderID,
		"amount", req.Amount,
		"currency", req.Currency,
		"transactionID", txnID)

	return json.Marshal(PaymentResponse{
		TransactionID: txnID,
		Status:        "approved",
		Message:       "Payment processed successfully",
	})
}

// ============================================================
// Order Module (Service Consumer - Depends on Inventory & Payment)
// ============================================================

type OrderModule struct {
	inventoryContainer mono.ServiceContainer
	paymentContainer   mono.ServiceContainer
	orderCounter       int
}

var (
	_ mono.Module                              = (*OrderModule)(nil)
	_ mono.ServiceProviderModule               = (*OrderModule)(nil)
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

// Dependencies declares which modules order depends on
func (m *OrderModule) Dependencies() []string {
	return []string{"inventory", "payment"}
}

// SetDependencyServiceContainer receives service containers from dependencies
func (m *OrderModule) SetDependencyServiceContainer(
	module string, container mono.ServiceContainer) {
	switch module {
	case "inventory":
		m.inventoryContainer = container
	case "payment":
		m.paymentContainer = container
	}
}

func (m *OrderModule) RegisterServices(container mono.ServiceContainer) error {
	return container.RegisterRequestReplyService(
		"create-order",
		m.handleCreateOrder,
	)
}

func (m *OrderModule) handleCreateOrder(
	ctx context.Context, msg *mono.Msg) ([]byte, error) {
	var req CreateOrderRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	m.orderCounter++
	orderID := fmt.Sprintf("ORD-%06d", m.orderCounter)

	slog.Info("Creating order",
		"orderID", orderID,
		"productID", req.ProductID,
		"quantity", req.Quantity)

	// Step 1: Check inventory
	var inventoryResp InventoryResponse
	err := helper.CallRequestReplyService(
		ctx,
		m.inventoryContainer,
		"check-stock",
		json.Marshal,
		json.Unmarshal,
		&InventoryRequest{ProductID: req.ProductID, Quantity: req.Quantity},
		&inventoryResp,
	)
	if err != nil {
		return nil, fmt.Errorf("inventory check failed: %w", err)
	}

	if !inventoryResp.Available {
		return json.Marshal(CreateOrderResponse{
			OrderID: orderID,
			Status:  "failed_out_of_stock",
			Message: fmt.Sprintf("Insufficient stock: %s", inventoryResp.Message),
		})
	}

	// Step 2: Process payment
	var paymentResp PaymentResponse
	err = helper.CallRequestReplyService(
		ctx,
		m.paymentContainer,
		"process-payment",
		json.Marshal,
		json.Unmarshal,
		&PaymentRequest{OrderID: orderID, Amount: req.Amount, Currency: req.Currency},
		&paymentResp,
	)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	if paymentResp.Status != "approved" {
		return json.Marshal(CreateOrderResponse{
			OrderID: orderID,
			Status:  "payment_failed",
			Message: paymentResp.Message,
		})
	}

	// Order successful
	slog.Info("Order created successfully",
		"orderID", orderID,
		"transactionID", paymentResp.TransactionID)

	return json.Marshal(CreateOrderResponse{
		OrderID:       orderID,
		Status:        "success",
		TransactionID: paymentResp.TransactionID,
		Message:       "Order created successfully",
	})
}

// ============================================================
// Order Service Adapter (Type-Safe Client)
// ============================================================

// OrderAdapter provides a type-safe interface to the order service
type OrderAdapter struct {
	container mono.ServiceContainer
}

func NewOrderAdapter(container mono.ServiceContainer) *OrderAdapter {
	return &OrderAdapter{container: container}
}

func (a *OrderAdapter) CreateOrder(
	ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	var response CreateOrderResponse
	err := helper.CallRequestReplyService(
		ctx,
		a.container,
		"create-order",
		json.Marshal,
		json.Unmarshal,
		req,
		&response,
	)
	return &response, err
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Multi-Module Example ===")
	fmt.Println("Demonstrates: Dependencies, RequestReply services, Service adapters")
	fmt.Println()

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
	inventoryModule := NewInventoryModule()
	paymentModule := NewPaymentModule()
	orderModule := NewOrderModule()

	// Register modules (framework resolves dependency order automatically)
	// Even though we register order first, framework will start inventory and payment first
	modules := []mono.Module{orderModule, inventoryModule, paymentModule}
	for _, module := range modules {
		if err := app.Register(module); err != nil {
			log.Fatalf("Failed to register module %s: %v", module.Name(), err)
		}
	}

	fmt.Printf("Registered modules: %v\n", app.Modules())

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Println()

	// Wait for services to be ready
	time.Sleep(100 * time.Millisecond)

	// Get order service container for external access
	orderContainer := app.Services("order")
	if orderContainer == nil {
		log.Fatal("Order service container not available")
	}

	// Create type-safe adapter
	orderAdapter := NewOrderAdapter(orderContainer)

	// Test scenarios
	fmt.Println("=== Running Test Scenarios ===")
	fmt.Println()

	// Scenario 1: Successful order
	fmt.Println("[Scenario 1] Successful Order (laptop)")
	resp, err := orderAdapter.CreateOrder(ctx, &CreateOrderRequest{
		ProductID: "laptop",
		Quantity:  2,
		Amount:    1999.98,
		Currency:  "USD",
	})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Result: %s - %s (TXN: %s)\n", resp.OrderID, resp.Status, resp.TransactionID)
	}
	fmt.Println()

	// Scenario 2: Out of stock
	fmt.Println("[Scenario 2] Out of Stock (laptop x 100)")
	resp, err = orderAdapter.CreateOrder(ctx, &CreateOrderRequest{
		ProductID: "laptop",
		Quantity:  100,
		Amount:    99999.00,
		Currency:  "USD",
	})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Result: %s - %s\n", resp.OrderID, resp.Status)
		fmt.Printf("  Message: %s\n", resp.Message)
	}
	fmt.Println()

	// Scenario 3: Product not found
	fmt.Println("[Scenario 3] Unknown Product")
	resp, err = orderAdapter.CreateOrder(ctx, &CreateOrderRequest{
		ProductID: "unknown-product",
		Quantity:  1,
		Amount:    99.99,
		Currency:  "USD",
	})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Result: %s - %s\n", resp.OrderID, resp.Status)
		fmt.Printf("  Message: %s\n", resp.Message)
	}
	fmt.Println()

	// Scenario 4: Another successful order
	fmt.Println("[Scenario 4] Successful Order (mouse)")
	resp, err = orderAdapter.CreateOrder(ctx, &CreateOrderRequest{
		ProductID: "mouse",
		Quantity:  5,
		Amount:    149.95,
		Currency:  "USD",
	})
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Result: %s - %s (TXN: %s)\n", resp.OrderID, resp.Status, resp.TransactionID)
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
