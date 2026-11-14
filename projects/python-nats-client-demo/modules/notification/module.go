package notification

import (
	"context"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/types"

	"github.com/example/python-nats-client-demo/modules/notification/contract"
)

// QueueGroup is the queue group name for email workers. It is a provider-side
// detail: senders only need the contract's subject.
const QueueGroup = "email-workers"

// NotificationModule provides notification services via QueueGroupService.
// This demonstrates fire-and-forget messaging where multiple workers
// process email jobs from a queue.
type NotificationModule struct{}

// Compile-time interface checks.
var (
	_ mono.Module                = (*NotificationModule)(nil)
	_ mono.ServiceProviderModule = (*NotificationModule)(nil)
)

// NewModule creates a new NotificationModule.
func NewModule() *NotificationModule {
	return &NotificationModule{}
}

// Name returns the module name.
func (m *NotificationModule) Name() string {
	return "notification"
}

// RegisterServices binds the typed handler to the module's own contract.
// The NATS subject is services.notification.email-send; messages are
// load-balanced across workers in the "email-workers" queue group.
func (m *NotificationModule) RegisterServices(container mono.ServiceContainer) error {
	err := contract.SendEmail.Register(container, types.TypedQGHP[contract.SendEmailRequest]{
		QueueGroup: QueueGroup,
		Handler:    m.handleEmailSend,
	})
	if err != nil {
		return fmt.Errorf("failed to register queue group service: %w", err)
	}

	log.Printf("[notification] Registered QueueGroupService: %s (queue: %s)", contract.SendEmail.Subject(), QueueGroup)
	return nil
}

// Start initializes the notification module.
func (m *NotificationModule) Start(_ context.Context) error {
	log.Println("[notification] Module started successfully")
	return nil
}

// Stop gracefully stops the notification module.
func (m *NotificationModule) Stop(_ context.Context) error {
	log.Println("[notification] Module stopped")
	return nil
}
