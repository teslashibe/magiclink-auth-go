package magiclink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSendInvalidJSON(t *testing.T) {
	svc := New(testConfig(), &mockCodeStore{}, &mockUserStore{}, &mockEmailSender{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	svc.HandleSend(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleVerifyCodeSuccess(t *testing.T) {
	codes := &mockCodeStore{}
	users := &mockUserStore{upsertID: "user-100", getID: "user-100", getName: "User"}
	svc := New(testConfig(), codes, users, &mockEmailSender{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/verify", strings.NewReader(`{"email":"user@example.com","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	svc.HandleVerifyCode(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got AuthResult
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.UserID != "user-100" || got.JWT == "" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestHandleVerifyLinkSuccess(t *testing.T) {
	cfg := testConfig()
	cfg.DeepLinkURL = "example://auth"

	codes := &mockCodeStore{
		lookupEmail: "user@example.com",
		lookupCode:  "123456",
	}
	users := &mockUserStore{upsertID: "user-101", getID: "user-101", getName: "User"}
	svc := New(cfg, codes, users, &mockEmailSender{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=abc", nil)
	rr := httptest.NewRecorder()

	svc.HandleVerifyLink(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(rr.Body.String(), "You're in") {
		t.Fatalf("response does not look like success HTML: %s", rr.Body.String())
	}
}
