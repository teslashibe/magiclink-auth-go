package ses

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type sendEmailAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// Sender sends email via AWS SES v2.
type Sender struct {
	client               sendEmailAPI
	fromAddress          string
	configurationSetName string
}

// Option configures Sender.
type Option func(*Sender)

// WithConfigurationSet sets an optional SES configuration set.
func WithConfigurationSet(name string) Option {
	return func(s *Sender) {
		s.configurationSetName = strings.TrimSpace(name)
	}
}

// New creates an SES sender from an SES client.
func New(client *sesv2.Client, fromAddress string, opts ...Option) *Sender {
	s := &Sender{
		client:      client,
		fromAddress: strings.TrimSpace(fromAddress),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewFromConfig creates an SES sender from an AWS config.
func NewFromConfig(cfg aws.Config, fromAddress string, opts ...Option) *Sender {
	return New(sesv2.NewFromConfig(cfg), fromAddress, opts...)
}

// Send sends one HTML email.
func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("ses client is required")
	}
	if strings.TrimSpace(s.fromAddress) == "" {
		return fmt.Errorf("ses from address is required")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.fromAddress),
		Destination: &sestypes.Destination{
			ToAddresses: []string{strings.TrimSpace(to)},
		},
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Subject: &sestypes.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &sestypes.Body{
					Html: &sestypes.Content{
						Data:    aws.String(htmlBody),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}
	if s.configurationSetName != "" {
		input.ConfigurationSetName = aws.String(s.configurationSetName)
	}

	if _, err := s.client.SendEmail(ctx, input); err != nil {
		return fmt.Errorf("ses send email: %w", err)
	}
	return nil
}
