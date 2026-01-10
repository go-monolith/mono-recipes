// Example: Using Built-in Middleware Modules
//
// This example demonstrates:
// - Using requestid middleware for request tracing
// - Using accesslog middleware for request logging
// - Using audit middleware for tamper-evident audit trails
// - Proper middleware registration order
// - Accessing request ID in handlers

package middlewareusecases

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
	"github.com/go-monolith/mono/middleware/accesslog"
	"github.com/go-monolith/mono/middleware/audit"
	"github.com/go-monolith/mono/middleware/requestid"
	"github.com/go-monolith/mono/pkg/types"
)

// ============================================================
// Order Processing Module
// ============================================================

type OrderModule struct {
	container types.ServiceContainer
}

var _ mono.ServiceProviderModule = (*OrderModule)(nil)

func NewOrderModule() *OrderModule {
	return &OrderModule{}
}

func (m *OrderModule) Name() string { return "orders" }

func (m *OrderModule) Start(_ context.Context) error {
	slog.Info("Order module started")
	return nil
}

func (m *OrderModule) Stop(_ context.Context) error {
	slog.Info("Order module stopped")
	return nil
}

func (m *OrderModule) RegisterServices(container types.ServiceContainer) error {
	m.container = container

	// Register order creation service
	if err := container.RegisterRequestReplyService("create-order", m.handleCreateOrder); err != nil {
		return err
	}

	// Register order status service
	return container.RegisterRequestReplyService("get-order", m.handleGetOrder)
}

type CreateOrderRequest struct {
	CustomerID string  `json:"customer_id"`
	Items      []Item  `json:"items"`
	Total      float64 `json:"total"`
}

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderResponse struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
}

func (m *OrderModule) handleCreateOrder(ctx context.Context, req *types.Msg) ([]byte, error) {
	// Extract request ID from context (injected by requestid middleware)
	reqID := requestid.GetRequestID(ctx)

	slog.Info("Processing order creation",
		"request_id", reqID,
		"data_size", len(req.Data))

	var orderReq CreateOrderRequest
	if err := json.Unmarshal(req.Data, &orderReq); err != nil {
		return nil, fmt.Errorf("invalid order request: %w", err)
	}

	// Simulate order creation
	orderID := fmt.Sprintf("ORD-%d", time.Now().UnixNano())

	response := CreateOrderResponse{
		OrderID:   orderID,
		Status:    "created",
		RequestID: reqID, // Include request ID in response for tracing
	}

	slog.Info("Order created",
		"request_id", reqID,
		"order_id", orderID,
		"customer_id", orderReq.CustomerID,
		"total", orderReq.Total)

	return json.Marshal(response)
}

type GetOrderRequest struct {
	OrderID string `json:"order_id"`
}

type GetOrderResponse struct {
	OrderID   string  `json:"order_id"`
	Status    string  `json:"status"`
	Total     float64 `json:"total"`
	RequestID string  `json:"request_id"`
}

func (m *OrderModule) handleGetOrder(ctx context.Context, req *types.Msg) ([]byte, error) {
	reqID := requestid.GetRequestID(ctx)

	var getReq GetOrderRequest
	if err := json.Unmarshal(req.Data, &getReq); err != nil {
		return nil, fmt.Errorf("invalid get order request: %w", err)
	}

	slog.Info("Fetching order",
		"request_id", reqID,
		"order_id", getReq.OrderID)

	// Simulate order lookup
	response := GetOrderResponse{
		OrderID:   getReq.OrderID,
		Status:    "processing",
		Total:     99.99,
		RequestID: reqID,
	}

	return json.Marshal(response)
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework Built-in Middleware Example ===")
	fmt.Println("Demonstrates: requestid, accesslog, and audit middleware")
	fmt.Println()

	// Create log files
	accessLogFile, err := os.OpenFile("/tmp/access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to create access log file: %v", err)
	}
	defer accessLogFile.Close()

	auditLogFile, err := os.OpenFile("/tmp/audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("Failed to create audit log file: %v", err)
	}
	defer auditLogFile.Close()

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// ============================================================
	// Register Middleware in Order (IMPORTANT!)
	// ============================================================

	// 1. Audit middleware FIRST - observes all events including other middleware
	auditMiddleware, err := audit.New(
		audit.WithOutput(auditLogFile),
		audit.WithHashChaining(""), // Start new chain
		audit.WithUserContext(func(ctx context.Context) string {
			// Extract user from context if available
			if user, ok := ctx.Value("user").(string); ok {
				return user
			}
			return "system"
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create audit middleware: %v", err)
	}
	if err := app.Register(auditMiddleware); err != nil {
		log.Fatalf("Failed to register audit middleware: %v", err)
	}
	fmt.Println("1. Audit middleware registered (observes all events)")

	// 2. Request ID middleware - injects request IDs for tracing
	requestIDMiddleware, err := requestid.New(
		requestid.WithHeaderName("X-Request-ID"),
	)
	if err != nil {
		log.Fatalf("Failed to create requestid middleware: %v", err)
	}
	if err := app.Register(requestIDMiddleware); err != nil {
		log.Fatalf("Failed to register requestid middleware: %v", err)
	}
	fmt.Println("2. RequestID middleware registered (injects request IDs)")

	// 3. Access log middleware - logs requests with timing (uses request IDs)
	accessLogMiddleware, err := accesslog.New(
		accesslog.WithOutput(accessLogFile),
		accesslog.WithFormat(accesslog.FormatJSON),
		accesslog.WithFields([]accesslog.Field{
			accesslog.FieldTimestamp,
			accesslog.FieldRequestID,
			accesslog.FieldModule,
			accesslog.FieldService,
			accesslog.FieldServiceType,
			accesslog.FieldDurationMS,
			accesslog.FieldStatus,
			accesslog.FieldRequestSize,
			accesslog.FieldResponseSize,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create accesslog middleware: %v", err)
	}
	if err := app.Register(accessLogMiddleware); err != nil {
		log.Fatalf("Failed to register accesslog middleware: %v", err)
	}
	fmt.Println("3. AccessLog middleware registered (logs requests)")

	// ============================================================
	// Register Application Modules AFTER Middleware
	// ============================================================

	orderModule := NewOrderModule()
	if err := app.Register(orderModule); err != nil {
		log.Fatalf("Failed to register order module: %v", err)
	}
	fmt.Println("4. Order module registered")

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("\nApp started successfully")
	fmt.Println()

	// ============================================================
	// Demo: Making Requests
	// ============================================================

	fmt.Println("=== Making Sample Requests ===")
	fmt.Println()

	// Get service client
	createOrderClient, err := orderModule.container.GetRequestReplyService("create-order")
	if err != nil {
		log.Fatalf("Failed to get create-order service: %v", err)
	}

	getOrderClient, err := orderModule.container.GetRequestReplyService("get-order")
	if err != nil {
		log.Fatalf("Failed to get get-order service: %v", err)
	}

	// Create an order
	createReq := CreateOrderRequest{
		CustomerID: "CUST-123",
		Items: []Item{
			{ProductID: "PROD-001", Quantity: 2},
			{ProductID: "PROD-002", Quantity: 1},
		},
		Total: 149.99,
	}
	createReqData, _ := json.Marshal(createReq)

	fmt.Println("Creating order...")
	resp, err := createOrderClient.Call(ctx, createReqData)
	if err != nil {
		fmt.Printf("Error creating order: %v\n", err)
	} else {
		var createResp CreateOrderResponse
		json.Unmarshal(resp, &createResp)
		fmt.Printf("Order created: %s (Request ID: %s)\n", createResp.OrderID, createResp.RequestID)

		// Get the order
		getReq := GetOrderRequest{OrderID: createResp.OrderID}
		getReqData, _ := json.Marshal(getReq)

		fmt.Println("\nFetching order...")
		resp, err = getOrderClient.Call(ctx, getReqData)
		if err != nil {
			fmt.Printf("Error getting order: %v\n", err)
		} else {
			var getResp GetOrderResponse
			json.Unmarshal(resp, &getResp)
			fmt.Printf("Order status: %s, Total: $%.2f (Request ID: %s)\n",
				getResp.Status, getResp.Total, getResp.RequestID)
		}
	}

	// Make a few more requests for access log demonstration
	fmt.Println("\nMaking additional requests for logging demo...")
	for i := 0; i < 3; i++ {
		createReq.CustomerID = fmt.Sprintf("CUST-%d", i+200)
		createReq.Total = float64(50 + i*25)
		data, _ := json.Marshal(createReq)
		createOrderClient.Call(ctx, data)
	}
	fmt.Println("Additional requests completed")

	fmt.Println()
	fmt.Println("=== Log Files Created ===")
	fmt.Println("Access log: /tmp/access.log (JSON format)")
	fmt.Println("Audit log:  /tmp/audit.log (with hash chaining)")

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
	fmt.Println()

	// Show sample log entries
	fmt.Println("=== Sample Access Log Entry ===")
	fmt.Println(`{"timestamp":"2024-01-15T10:30:00Z","request_id":"abc-123","module":"orders",`)
	fmt.Println(`"service":"create-order","service_type":"request_reply","duration_ms":5,`)
	fmt.Println(`"status":"success","request_size":150,"response_size":80}`)
	fmt.Println()

	fmt.Println("=== Sample Audit Log Entry ===")
	fmt.Println(`{"timestamp":"2024-01-15T10:30:00Z","event_type":"service.registered",`)
	fmt.Println(`"module_name":"orders","service_name":"create-order",`)
	fmt.Println(`"details":{"service_type":"request_reply"},`)
	fmt.Println(`"prev_hash":"...","entry_hash":"..."}`)
}
