// Package contract is the published port of the payment module: the DTOs of
// its services and the typed contracts they are registered, published and
// called through. It is the only package of the payment module that other code
// may import.
//
// The subjects services.payment.payment-process (JetStream) and
// services.payment.status (request-reply) are what the Python PaymentClient
// uses, so module and service names must not change.
package contract

import "github.com/go-monolith/mono/pkg/helper"

// PaymentStatus is the status of a payment.
type PaymentStatus string

// Payment statuses.
const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
)

// PaymentRequest is a payment submitted to the payment-process stream.
type PaymentRequest struct {
	PaymentID      string  `json:"payment_id"`
	UserID         string  `json:"user_id"`
	SubscriptionID string  `json:"subscription_id"`
	Amount         float64 `json:"amount"`
}

// StatusRequest asks for the status of a payment.
type StatusRequest struct {
	PaymentID string `json:"payment_id"`
}

// StatusResponse reports the status of a payment.
type StatusResponse struct {
	PaymentID string        `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
}

// ProcessPayment is the durable stream consumer service on
// services.payment.payment-process. Publishers (Python's js.publish, or
// ProcessPayment.Publish in Go) get a JetStream acknowledgement.
var ProcessPayment = helper.StreamConsumerService[PaymentRequest]("payment", "payment-process")

// Status is the request-reply service on services.payment.status.
var Status = helper.RequestReplyService[StatusRequest, StatusResponse]("payment", "status")
