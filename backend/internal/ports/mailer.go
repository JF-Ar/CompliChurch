package ports

import "context"

// Mailer is the port for sending transactional email.
// All sends are fire-and-forget; the caller does not wait for delivery confirmation.
type Mailer interface {
	Send(ctx context.Context, msg EmailMessage) error
}

type EmailMessage struct {
	To       string
	Template string // template key, e.g. "member_welcome"
	Data     map[string]any
}
