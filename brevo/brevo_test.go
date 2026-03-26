package brevo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSenderSendSuccess(t *testing.T) {
	var gotAPIKey string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("api-key")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	s := New("api-key", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	if err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAPIKey != "api-key" {
		t.Fatalf("api-key header = %q", gotAPIKey)
	}
	if gotPayload["htmlContent"] != "<p>html</p>" {
		t.Fatalf("request payload missing html content: %#v", gotPayload)
	}
}

func TestSenderSendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad key"}`))
	}))
	defer server.Close()

	s := New("bad-key", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>")
	if err == nil {
		t.Fatalf("Send() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "brevo 401") {
		t.Fatalf("unexpected error: %v", err)
	}
}
