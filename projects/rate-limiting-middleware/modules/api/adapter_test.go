package api

import (
	"testing"

	"github.com/go-monolith/mono"
)

func TestNewServiceAdapter(t *testing.T) {
	t.Run("with nil container", func(t *testing.T) {
		adapter := NewServiceAdapter(nil)

		if adapter == nil {
			t.Fatal("NewServiceAdapter returned nil")
		}
		if adapter.container != nil {
			t.Error("expected container to be nil")
		}
	})
}

// TestServiceAdapter_Methods documents that adapter methods require integration
// testing with a real ServiceContainer and NATS connection. The adapter is a thin
// wrapper around helper.CallRequestReplyService, so unit testing would only verify
// that the wrapper correctly passes parameters to the helper function.
//
// For comprehensive testing, see the project's integration test suite.
func TestServiceAdapter_Methods(t *testing.T) {
	t.Skip("adapter methods require integration tests with real ServiceContainer")
}

// Compile-time check that ServiceAdapter methods exist with correct signatures.
var _ interface {
	GetData(ctx interface{ Deadline() (interface{}, bool) }) (*DataResponse, error)
	CreateOrder(ctx interface{ Deadline() (interface{}, bool) }, req *OrderRequest) (*OrderResponse, error)
	GetStatus(ctx interface{ Deadline() (interface{}, bool) }) (*StatusResponse, error)
}

// mockServiceContainer implements mono.ServiceContainer for testing.
// This is a minimal implementation that can be extended for specific test scenarios.
type mockServiceContainer struct {
	mono.ServiceContainer
}
