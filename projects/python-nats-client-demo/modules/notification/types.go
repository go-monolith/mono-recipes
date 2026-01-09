package notification

// SendEmailRequest is the request for sending an email via QueueGroupService.
type SendEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}
