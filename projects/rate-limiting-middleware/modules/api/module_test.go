package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-monolith/mono/pkg/types"
)

// mockLogger implements types.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...any) {}
func (m *mockLogger) Info(_ string, _ ...any)  {}
func (m *mockLogger) Warn(_ string, _ ...any)  {}
func (m *mockLogger) Error(_ string, _ ...any) {}
func (m *mockLogger) With(_ ...any) types.Logger {
	return m
}
func (m *mockLogger) WithModule(_ string) types.Logger {
	return m
}
func (m *mockLogger) WithError(_ error) types.Logger {
	return m
}

func newMockLogger() types.Logger {
	return &mockLogger{}
}

func TestNewModule(t *testing.T) {
	m := NewModule(newMockLogger())

	if m == nil {
		t.Fatal("NewModule returned nil")
	}
	if m.logger == nil {
		t.Error("expected logger to be set")
	}
}

func TestModule_Name(t *testing.T) {
	m := NewModule(newMockLogger())

	if name := m.Name(); name != "api" {
		t.Errorf("Name() = %q, want 'api'", name)
	}
}

func TestModule_StartStop(t *testing.T) {
	m := NewModule(newMockLogger())
	ctx := context.Background()

	// Start should work
	if err := m.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// startTime should be set
	if m.startTime.IsZero() {
		t.Error("expected startTime to be set after Start()")
	}

	// Stop should work
	if err := m.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestModule_handleGetData(t *testing.T) {
	m := NewModule(newMockLogger())
	ctx := context.Background()
	_ = m.Start(ctx)

	msg := &types.Msg{}
	resp, err := m.handleGetData(ctx, msg)

	if err != nil {
		t.Fatalf("handleGetData() error = %v", err)
	}

	var data DataResponse
	if err := json.Unmarshal(resp, &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if data.ID == "" {
		t.Error("expected non-empty ID")
	}
	if data.Name != "Sample Data" {
		t.Errorf("expected Name 'Sample Data', got %q", data.Name)
	}
	if data.Value != 42 {
		t.Errorf("expected Value 42, got %d", data.Value)
	}
	if data.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

func TestModule_handleCreateOrder(t *testing.T) {
	m := NewModule(newMockLogger())
	ctx := context.Background()
	_ = m.Start(ctx)

	tests := []struct {
		name        string
		request     OrderRequest
		wantErr     bool
		errContains string
	}{
		{
			name: "valid order",
			request: OrderRequest{
				ProductID: "prod-123",
				Quantity:  2,
				Price:     29.99,
			},
			wantErr: false,
		},
		{
			name: "missing product ID",
			request: OrderRequest{
				ProductID: "",
				Quantity:  1,
				Price:     10.00,
			},
			wantErr:     true,
			errContains: "product_id is required",
		},
		{
			name: "zero quantity",
			request: OrderRequest{
				ProductID: "prod-123",
				Quantity:  0,
				Price:     10.00,
			},
			wantErr:     true,
			errContains: "quantity must be positive",
		},
		{
			name: "negative quantity",
			request: OrderRequest{
				ProductID: "prod-123",
				Quantity:  -1,
				Price:     10.00,
			},
			wantErr:     true,
			errContains: "quantity must be positive",
		},
		{
			name: "zero price",
			request: OrderRequest{
				ProductID: "prod-123",
				Quantity:  1,
				Price:     0,
			},
			wantErr:     true,
			errContains: "price must be positive",
		},
		{
			name: "negative price",
			request: OrderRequest{
				ProductID: "prod-123",
				Quantity:  1,
				Price:     -10.00,
			},
			wantErr:     true,
			errContains: "price must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqData, _ := json.Marshal(tt.request)
			msg := &types.Msg{Data: reqData}

			resp, err := m.handleCreateOrder(ctx, msg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errContains != "" && err.Error() != tt.errContains {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("handleCreateOrder() error = %v", err)
			}

			var order OrderResponse
			if err := json.Unmarshal(resp, &order); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if order.OrderID == "" {
				t.Error("expected non-empty OrderID")
			}
			if order.ProductID != tt.request.ProductID {
				t.Errorf("expected ProductID %q, got %q", tt.request.ProductID, order.ProductID)
			}
			if order.Quantity != tt.request.Quantity {
				t.Errorf("expected Quantity %d, got %d", tt.request.Quantity, order.Quantity)
			}
			expectedTotal := float64(tt.request.Quantity) * tt.request.Price
			if order.Total != expectedTotal {
				t.Errorf("expected Total %.2f, got %.2f", expectedTotal, order.Total)
			}
			if order.Status != "pending" {
				t.Errorf("expected Status 'pending', got %q", order.Status)
			}
			if order.CreatedAt.IsZero() {
				t.Error("expected non-zero CreatedAt")
			}
		})
	}
}

func TestModule_handleCreateOrder_InvalidJSON(t *testing.T) {
	m := NewModule(newMockLogger())
	ctx := context.Background()
	_ = m.Start(ctx)

	msg := &types.Msg{Data: []byte("invalid json")}
	_, err := m.handleCreateOrder(ctx, msg)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestModule_handleGetStatus(t *testing.T) {
	m := NewModule(newMockLogger())
	ctx := context.Background()
	_ = m.Start(ctx)

	msg := &types.Msg{}
	resp, err := m.handleGetStatus(ctx, msg)

	if err != nil {
		t.Fatalf("handleGetStatus() error = %v", err)
	}

	var status StatusResponse
	if err := json.Unmarshal(resp, &status); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if status.Service != "rate-limiting-demo" {
		t.Errorf("expected Service 'rate-limiting-demo', got %q", status.Service)
	}
	if status.Status != "healthy" {
		t.Errorf("expected Status 'healthy', got %q", status.Status)
	}
	if status.Uptime == "" {
		t.Error("expected non-empty Uptime")
	}
	if status.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
}

func TestServiceConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"ServiceGetData", ServiceGetData, "api.getData"},
		{"ServiceCreateOrder", ServiceCreateOrder, "api.createOrder"},
		{"ServiceGetStatus", ServiceGetStatus, "api.getStatus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}
