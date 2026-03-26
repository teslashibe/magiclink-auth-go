package postmark

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
	var gotToken string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Postmark-Server-Token")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New("server-token", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	if err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotToken != "server-token" {
		t.Fatalf("postmark token header = %q", gotToken)
	}
	if gotPayload["HtmlBody"] != "<p>html</p>" {
		t.Fatalf("request payload missing html body: %#v", gotPayload)
	}
}

func TestSenderSendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ErrorCode":10,"Message":"Bad token"}`))
	}))
	defer server.Close()

	s := New("bad-token", "MyApp <noreply@example.com>", WithBaseURL(server.URL))
	err := s.Send(context.Background(), "user@example.com", "subject", "<p>html</p>")
	if err == nil {
		t.Fatalf("Send() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "postmark 401") {
		t.Fatalf("unexpected error: %v", err)
	}
}
