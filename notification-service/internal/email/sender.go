package email

import "context"

// #######################################
// EMAIL SENDER (Adapter Pattern)
// #######################################
//
// EmailSender is the port that the notification use case talks to.
// Implementations (adapters) are vendor-specific: SMTP, Mailjet,
// SendGrid, Simulated, etc. They live in the same package.
//
// The use case must NEVER import a concrete provider — only this
// interface. Provider selection happens once, at startup,
// based on the PROVIDER_MODE environment variable.

type EmailSender interface {
	// SendOrderConfirmation sends an "order paid" email.
	// Returns nil on success; any error means the worker should retry.
	SendOrderConfirmation(ctx context.Context, msg Message) error
}

// Message carries the data every provider needs to render the email.
// Kept minimal so providers stay easy to plug in.
type Message struct {
	To       string // recipient email
	OrderID  string
	AmountUS string // "$99.99" pre-formatted by the caller
}
