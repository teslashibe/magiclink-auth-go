package resend

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
	var gotAuth string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New("api-key", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	if err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotAuth != "Bearer api-key" {
		t.Fatalf("authorization header = %q", gotAuth)
	}
	if gotPayload["from"] != "MyApp <noreply@example.com>" {
		t.Fatalf("request body missing from address: %#v", gotPayload)
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
	if !strings.Contains(err.Error(), "resend 401") {
		t.Fatalf("unexpected error: %v", err)
	}
}
