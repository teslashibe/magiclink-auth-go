package magiclink

import "context"

// EmailSender delivers the final subject/body payload to a user.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// EmailRenderer renders the magic-link email payload.
type EmailRenderer interface {
	Render(code, magicURL, appName string) (subject, htmlBody string)
}
