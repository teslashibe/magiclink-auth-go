package mailgun

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSenderSendSuccess(t *testing.T) {
	var gotAuth string
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New("api-key", "mg.example.com", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	if err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("authorization header not set for basic auth: %q", gotAuth)
	}
	if !strings.Contains(gotBody, "subject=subject") {
		t.Fatalf("request payload missing subject: %s", gotBody)
	}
}

func TestSenderSendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	s := New("bad-key", "mg.example.com", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>")
	if err == nil {
		t.Fatalf("Send() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "mailgun 401") {
		t.Fatalf("unexpected error: %v", err)
	}
}
