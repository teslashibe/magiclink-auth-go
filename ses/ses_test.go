package ses

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

type mockSESClient struct {
	input *sesv2.SendEmailInput
	err   error
}

func (m *mockSESClient) SendEmail(_ context.Context, params *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	m.input = params
	if m.err != nil {
		return nil, m.err
	}
	return &sesv2.SendEmailOutput{}, nil
}

func TestSenderSendSuccess(t *testing.T) {
	client := &mockSESClient{}
	s := &Sender{
		client:      client,
		fromAddress: "noreply@example.com",
	}

	err := s.Send(context.Background(), "user@example.com", "Subject", "<p>Hello</p>")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if client.input == nil {
		t.Fatalf("SendEmail input was nil")
	}
	if got := *client.input.FromEmailAddress; got != "noreply@example.com" {
		t.Fatalf("from address = %q", got)
	}
	if got := client.input.Destination.ToAddresses[0]; got != "user@example.com" {
		t.Fatalf("to address = %q", got)
	}
}

func TestSenderSendError(t *testing.T) {
	client := &mockSESClient{err: errors.New("boom")}
	s := &Sender{
		client:      client,
		fromAddress: "noreply@example.com",
	}

	err := s.Send(context.Background(), "user@example.com", "Subject", "<p>Hello</p>")
	if err == nil {
		t.Fatalf("Send() error = nil, want error")
	}
}
