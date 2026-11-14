// Package contract is the published port of the notification module: the DTO
// of its queue group service and the typed contract it is registered and sent
// through. It is the only package of the notification module that other code
// may import.
//
// The subject services.notification.email-send is what the Python EmailClient
// publishes to, so module and service names must not change.
package contract

import "github.com/go-monolith/mono/pkg/helper"

// SendEmailRequest is the payload of the email-send queue group service.
type SendEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SendEmail is the fire-and-forget queue group service on
// services.notification.email-send.
var SendEmail = helper.QueueGroupService[SendEmailRequest]("notification", "email-send")
