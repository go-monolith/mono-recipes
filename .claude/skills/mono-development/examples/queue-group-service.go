// queue-group-service.go demonstrates QueueGroupService with QGHP pattern
package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-monolith/mono"
)

// Module provides notification services via QueueGroupService
type Module struct {
	logger mono.Logger
}

// Compile-time interface checks
var (
	_ mono.Module                = (*Module)(nil)
	_ mono.ServiceProviderModule = (*Module)(nil)
)

// NewModule creates a new notification module
func NewModule(logger mono.Logger) *Module {
	return &Module{
		logger: logger,
	}
}

// Name returns the module name
func (m *Module) Name() string { return "notification" }

// RegisterServices registers queue group services
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
	// Register email service with single worker queue
	if err := container.RegisterQueueGroupService(
		"email-send",
		mono.QGHP{
			QueueGroup: "email-workers",
			Handler:    m.handleEmailSend,
		},
	); err != nil {
		return fmt.Errorf("failed to register email-send service: %w", err)
	}

	// Register push notification with load-balanced workers
	if err := container.RegisterQueueGroupService(
		"push-send",
		mono.QGHP{QueueGroup: "push-worker-1", Handler: m.handlePushSend},
		mono.QGHP{QueueGroup: "push-worker-2", Handler: m.handlePushSend},
		mono.QGHP{QueueGroup: "push-worker-3", Handler: m.handlePushSend},
	); err != nil {
		return fmt.Errorf("failed to register push-send service: %w", err)
	}

	// Register SMS service with dedicated queue
	if err := container.RegisterQueueGroupService(
		"sms-send",
		mono.QGHP{
			QueueGroup: "sms-workers",
			Handler:    m.handleSMSSend,
		},
	); err != nil {
		return fmt.Errorf("failed to register sms-send service: %w", err)
	}

	log.Println("[notification] Registered queue group services: email-send, push-send, sms-send")
	return nil
}

// Start initializes the notification module
func (m *Module) Start(ctx context.Context) error {
	log.Println("[notification] Module started")
	return nil
}

// Stop cleans up the notification module
func (m *Module) Stop(ctx context.Context) error {
	log.Println("[notification] Module stopped")
	return nil
}

// Request/Response types
type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type PushRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type SMSRequest struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}

// Handler implementations (fire-and-forget, no response)

func (m *Module) handleEmailSend(ctx context.Context, msg *mono.Msg) error {
	var req EmailRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		m.logger.Error("invalid email request", "error", err)
		return err // Return error to trigger redelivery if using JetStream
	}

	// Validate
	if req.To == "" || req.Subject == "" {
		m.logger.Error("missing required fields", "to", req.To, "subject", req.Subject)
		return fmt.Errorf("missing required fields")
	}

	// Simulate email sending
	log.Printf("[notification] Sending email to %s: %s", req.To, req.Subject)
	time.Sleep(100 * time.Millisecond) // Simulate work

	m.logger.Info("email sent", "to", req.To, "subject", req.Subject)
	return nil
}

func (m *Module) handlePushSend(ctx context.Context, msg *mono.Msg) error {
	var req PushRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		m.logger.Error("invalid push request", "error", err)
		return err
	}

	// Validate
	if req.UserID == "" || req.Title == "" {
		return fmt.Errorf("missing required fields")
	}

	// Simulate push notification
	log.Printf("[notification] Sending push to user %s: %s", req.UserID, req.Title)
	time.Sleep(50 * time.Millisecond)

	m.logger.Info("push sent", "user_id", req.UserID, "title", req.Title)
	return nil
}

func (m *Module) handleSMSSend(ctx context.Context, msg *mono.Msg) error {
	var req SMSRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		m.logger.Error("invalid sms request", "error", err)
		return err
	}

	// Validate
	if req.PhoneNumber == "" || req.Message == "" {
		return fmt.Errorf("missing required fields")
	}

	// Simulate SMS sending
	log.Printf("[notification] Sending SMS to %s: %s", req.PhoneNumber, req.Message)
	time.Sleep(200 * time.Millisecond)

	m.logger.Info("sms sent", "phone", req.PhoneNumber)
	return nil
}

// Example: Publishing to queue group service from another module
/*
func (m *OrderModule) sendOrderConfirmation(ctx context.Context, order Order) error {
    // Publish to email queue (fire-and-forget)
    emailReq := EmailRequest{
        To:      order.CustomerEmail,
        Subject: fmt.Sprintf("Order Confirmation #%s", order.ID),
        Body:    fmt.Sprintf("Thank you for your order of $%.2f", order.Total),
    }

    data, _ := json.Marshal(emailReq)
    return m.nc.Publish("services.notification.email-send", data)
}
*/
