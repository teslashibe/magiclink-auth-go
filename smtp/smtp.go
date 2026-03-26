package smtp

import (
	"context"
	"fmt"
	"net"
	netsmtp "net/smtp"
	"strings"
	"time"
)

type sendMailFunc func(addr string, a netsmtp.Auth, from string, to []string, msg []byte) error

// Sender sends email via plain SMTP.
type Sender struct {
	addr        string
	username    string
	password    string
	fromAddress string
	sendMail    sendMailFunc
}

// Option configures Sender.
type Option func(*Sender)

// WithSendMail sets a custom sendmail function (primarily for tests).
func WithSendMail(fn sendMailFunc) Option {
	return func(s *Sender) {
		if fn != nil {
			s.sendMail = fn
		}
	}
}

// New creates a Sender.
func New(addr, username, password, fromAddress string, opts ...Option) *Sender {
	s := &Sender{
		addr:        strings.TrimSpace(addr),
		username:    strings.TrimSpace(username),
		password:    password,
		fromAddress: strings.TrimSpace(fromAddress),
		sendMail:    netsmtp.SendMail,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Send sends one HTML email.
func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error {
	if strings.TrimSpace(s.addr) == "" {
		return fmt.Errorf("smtp addr is required")
	}
	if strings.TrimSpace(s.fromAddress) == "" {
		return fmt.Errorf("smtp from address is required")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}
	if s.sendMail == nil {
		return fmt.Errorf("smtp sendmail function is required")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	host := smtpHost(s.addr)
	var auth netsmtp.Auth
	if s.username != "" || s.password != "" {
		auth = netsmtp.PlainAuth("", s.username, s.password, host)
	}

	msg := buildMIMEMessage(s.fromAddress, strings.TrimSpace(to), subject, htmlBody)
	if err := s.sendMail(s.addr, auth, s.fromAddress, []string{strings.TrimSpace(to)}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp sendmail: %w", err)
	}
	return nil
}

func smtpHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	parts := strings.Split(addr, ":")
	return parts[0]
}

func buildMIMEMessage(from, to, subject, htmlBody string) string {
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody
}
