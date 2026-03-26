package postmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.postmarkapp.com/email"

// Sender sends email via Postmark.
type Sender struct {
	serverToken   string
	fromAddress   string
	messageStream string
	client        *http.Client
	baseURL       string
}

// Option configures Sender.
type Option func(*Sender)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(s *Sender) {
		if client != nil {
			s.client = client
		}
	}
}

// WithBaseURL sets a custom endpoint (primarily for tests).
func WithBaseURL(baseURL string) Option {
	return func(s *Sender) {
		if strings.TrimSpace(baseURL) != "" {
			s.baseURL = strings.TrimSpace(baseURL)
		}
	}
}

// WithMessageStream sets an optional Postmark message stream.
func WithMessageStream(stream string) Option {
	return func(s *Sender) {
		s.messageStream = strings.TrimSpace(stream)
	}
}

// New creates a Sender.
func New(serverToken, fromAddress string, opts ...Option) *Sender {
	s := &Sender{
		serverToken: strings.TrimSpace(serverToken),
		fromAddress: strings.TrimSpace(fromAddress),
		client:      &http.Client{Timeout: 15 * time.Second},
		baseURL:     defaultBaseURL,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Send sends one HTML email.
func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error {
	if strings.TrimSpace(s.serverToken) == "" {
		return fmt.Errorf("postmark server token is required")
	}
	if strings.TrimSpace(s.fromAddress) == "" {
		return fmt.Errorf("postmark from address is required")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}

	payload := map[string]any{
		"From":     s.fromAddress,
		"To":       strings.TrimSpace(to),
		"Subject":  subject,
		"HtmlBody": htmlBody,
	}
	if s.messageStream != "" {
		payload["MessageStream"] = s.messageStream
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal postmark payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create postmark request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", s.serverToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("postmark request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("postmark %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}
