package payment

import "github.com/example/python-nats-client-demo/modules/payment/contract"

// PaymentResult is the stored result of a payment. It is internal to the
// module; callers see it through contract.StatusResponse.
type PaymentResult struct {
	PaymentID string                 `json:"payment_id"`
	Status    contract.PaymentStatus `json:"status"`
	Message   string                 `json:"message,omitempty"`
}
