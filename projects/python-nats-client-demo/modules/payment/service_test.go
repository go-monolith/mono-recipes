package payment

import (
	"context"
	"testing"
)

func TestPaymentStore(t *testing.T) {
	t.Run("set and get payment", func(t *testing.T) {
		store := newPaymentStore()

		result := &PaymentResult{
			PaymentID: "pay-001",
			Status:    PaymentStatusCompleted,
			Message:   "Success",
		}

		store.set(result)
		got := store.get("pay-001")

		if got == nil {
			t.Fatal("expected payment result, got nil")
		}

		if got.PaymentID != result.PaymentID {
			t.Errorf("PaymentID = %v, want %v", got.PaymentID, result.PaymentID)
		}

		if got.Status != result.Status {
			t.Errorf("Status = %v, want %v", got.Status, result.Status)
		}

		if got.Message != result.Message {
			t.Errorf("Message = %v, want %v", got.Message, result.Message)
		}
	})

	t.Run("get non-existent payment returns nil", func(t *testing.T) {
		store := newPaymentStore()

		got := store.get("non-existent")

		if got != nil {
			t.Errorf("expected nil for non-existent payment, got %v", got)
		}
	})

	t.Run("update existing payment", func(t *testing.T) {
		store := newPaymentStore()

		// Initial state
		store.set(&PaymentResult{
			PaymentID: "pay-002",
			Status:    PaymentStatusProcessing,
			Message:   "Processing",
		})

		// Update to completed
		store.set(&PaymentResult{
			PaymentID: "pay-002",
			Status:    PaymentStatusCompleted,
			Message:   "Done",
		})

		got := store.get("pay-002")
		if got.Status != PaymentStatusCompleted {
			t.Errorf("Status = %v, want %v", got.Status, PaymentStatusCompleted)
		}
		if got.Message != "Done" {
			t.Errorf("Message = %v, want %v", got.Message, "Done")
		}
	})
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(*paymentStore)
		paymentID   string
		wantStatus  PaymentStatus
		wantMessage string
	}{
		{
			name:        "payment not found",
			setupStore:  func(s *paymentStore) {},
			paymentID:   "unknown",
			wantStatus:  PaymentStatusPending,
			wantMessage: "Payment not found or not yet processed",
		},
		{
			name: "payment completed",
			setupStore: func(s *paymentStore) {
				s.set(&PaymentResult{
					PaymentID: "pay-001",
					Status:    PaymentStatusCompleted,
					Message:   "Payment processed successfully",
				})
			},
			paymentID:   "pay-001",
			wantStatus:  PaymentStatusCompleted,
			wantMessage: "Payment processed successfully",
		},
		{
			name: "payment processing",
			setupStore: func(s *paymentStore) {
				s.set(&PaymentResult{
					PaymentID: "pay-002",
					Status:    PaymentStatusProcessing,
					Message:   "",
				})
			},
			paymentID:   "pay-002",
			wantStatus:  PaymentStatusProcessing,
			wantMessage: "",
		},
		{
			name: "payment failed",
			setupStore: func(s *paymentStore) {
				s.set(&PaymentResult{
					PaymentID: "pay-003",
					Status:    PaymentStatusFailed,
					Message:   "Insufficient funds",
				})
			},
			paymentID:   "pay-003",
			wantStatus:  PaymentStatusFailed,
			wantMessage: "Insufficient funds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newPaymentStore()
			tt.setupStore(store)

			m := &PaymentModule{store: store}

			resp, err := m.getStatus(context.Background(), StatusRequest{PaymentID: tt.paymentID}, nil)

			if err != nil {
				t.Fatalf("getStatus() returned error: %v", err)
			}

			if resp.PaymentID != tt.paymentID {
				t.Errorf("PaymentID = %v, want %v", resp.PaymentID, tt.paymentID)
			}

			if resp.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", resp.Status, tt.wantStatus)
			}

			if resp.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", resp.Message, tt.wantMessage)
			}
		})
	}
}

func TestPaymentStatusConstants(t *testing.T) {
	// Verify status constants match expected values
	statuses := map[PaymentStatus]string{
		PaymentStatusPending:    "pending",
		PaymentStatusProcessing: "processing",
		PaymentStatusCompleted:  "completed",
		PaymentStatusFailed:     "failed",
	}

	for status, expected := range statuses {
		if string(status) != expected {
			t.Errorf("PaymentStatus %v = %q, want %q", status, string(status), expected)
		}
	}
}
