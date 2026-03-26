package smtp

import (
	"context"
	netsmtp "net/smtp"
	"strings"
	"testing"
)

func TestSenderSendSuccess(t *testing.T) {
	var gotAddr string
	var gotFrom string
	var gotTo []string
	var gotMsg string

	s := New(
		"smtp.example.com:587",
		"user",
		"pass",
		"MyApp <noreply@example.com>",
		WithSendMail(func(addr string, a netsmtp.Auth, from string, to []string, msg []byte) error {
			gotAddr = addr
			gotFrom = from
			gotTo = to
			gotMsg = string(msg)
			return nil
		}),
	)

	if err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAddr != "smtp.example.com:587" {
		t.Fatalf("addr = %q", gotAddr)
	}
	if gotFrom != "MyApp <noreply@example.com>" {
		t.Fatalf("from = %q", gotFrom)
	}
	if len(gotTo) != 1 || gotTo[0] != "user@example.com" {
		t.Fatalf("to = %#v", gotTo)
	}
	if !strings.Contains(gotMsg, "Subject: subject") {
		t.Fatalf("message missing subject: %s", gotMsg)
	}
}

func TestSenderContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := New("smtp.example.com:587", "", "", "noreply@example.com")
	err := s.Send(ctx, "user@example.com", "subject", "<p>html</p>")
	if err == nil {
		t.Fatalf("Send() error = nil, want error")
	}
}
