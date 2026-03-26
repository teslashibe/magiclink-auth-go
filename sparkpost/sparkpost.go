package sparkpost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.sparkpost.com/api/v1/transmissions"

// Sender sends email via SparkPost.
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

// WithBaseURL sets a custom endpoint (primarily for tests).
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
		return fmt.Errorf("sparkpost api key is required")
	}
	if strings.TrimSpace(s.fromAddress) == "" {
		return fmt.Errorf("sparkpost from address is required")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}

	fromEmail, fromName, err := parseFromAddress(s.fromAddress)
	if err != nil {
		return fmt.Errorf("invalid sparkpost from address: %w", err)
	}

	from := map[string]string{"email": fromEmail}
	if fromName != "" {
		from["name"] = fromName
	}

	payload := map[string]any{
		"content": map[string]any{
			"from":    from,
			"subject": subject,
			"html":    htmlBody,
		},
		"recipients": []any{
			map[string]any{
				"address": map[string]string{
					"email": strings.TrimSpace(to),
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sparkpost payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sparkpost request: %w", err)
	}
	req.Header.Set("Authorization", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sparkpost request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sparkpost %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func parseFromAddress(fromAddress string) (email, name string, err error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(fromAddress))
	if err != nil {
		return "", "", err
	}
	return addr.Address, addr.Name, nil
}
