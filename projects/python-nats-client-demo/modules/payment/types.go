package payment

// PaymentStatus represents the status of a payment.
type PaymentStatus string

// Payment statuses.
const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
)

// PaymentRequest is the request to process a payment via StreamConsumerService.
type PaymentRequest struct {
	PaymentID      string  `json:"payment_id"`
	UserID         string  `json:"user_id"`
	SubscriptionID string  `json:"subscription_id"`
	Amount         float64 `json:"amount"`
}

// PaymentResult is the stored result of a payment.
type PaymentResult struct {
	PaymentID string        `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
}

// StatusRequest is the request to get payment status via RequestReplyService.
type StatusRequest struct {
	PaymentID string `json:"payment_id"`
}

// StatusResponse is the response for payment status queries.
type StatusResponse struct {
	PaymentID string        `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	Message   string        `json:"message,omitempty"`
}
