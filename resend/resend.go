package resend

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

const defaultBaseURL = "https://api.resend.com/emails"

// Sender sends email via Resend.
type Sender struct {
	apiKey      string
	fromAddress string
	client      *http.Client
	baseURL     string
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

// WithBaseURL sets a custom Resend endpoint (primarily for tests).
func WithBaseURL(baseURL string) Option {
	return func(s *Sender) {
		if strings.TrimSpace(baseURL) != "" {
			s.baseURL = strings.TrimSpace(baseURL)
		}
	}
}

// New creates a Sender.
func New(apiKey, fromAddress string, opts ...Option) *Sender {
	s := &Sender{
		apiKey:      strings.TrimSpace(apiKey),
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
	if strings.TrimSpace(s.apiKey) == "" {
		return fmt.Errorf("resend api key is required")
	}
	if strings.TrimSpace(s.fromAddress) == "" {
		return fmt.Errorf("resend from address is required")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}

	payload := map[string]any{
		"from":    s.fromAddress,
		"to":      []string{strings.TrimSpace(to)},
		"subject": subject,
		"html":    htmlBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}
