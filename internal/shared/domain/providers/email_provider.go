package providers

import "context"

// EmailMessage represents an email to be sent using a named template.
type EmailMessage struct {
	To           string
	Subject      string
	TemplateName string
	Data         map[string]any
}

// EmailProvider defines the contract for sending emails.
type EmailProvider interface {
	Send(ctx context.Context, msg EmailMessage) error
}
