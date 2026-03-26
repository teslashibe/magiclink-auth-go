package mailgun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.mailgun.net/v3"

// Sender sends email via Mailgun.
type Sender struct {
	apiKey      string
	domain      string
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
func New(apiKey, domain, fromAddress string, opts ...Option) *Sender {
	s := &Sender{
		apiKey:      strings.TrimSpace(apiKey),
		domain:      strings.TrimSpace(domain),
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
		return fmt.Errorf("mailgun api key is required")
	}
	if strings.TrimSpace(s.domain) == "" {
		return fmt.Errorf("mailgun domain is required")
	}
	if strings.TrimSpace(s.fromAddress) == "" {
		return fmt.Errorf("mailgun from address is required")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient email is required")
	}

	form := url.Values{
		"from":    {s.fromAddress},
		"to":      {strings.TrimSpace(to)},
		"subject": {subject},
		"html":    {htmlBody},
	}

	endpoint := strings.TrimRight(s.baseURL, "/") + "/" + s.domain + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create mailgun request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailgun request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailgun %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}
