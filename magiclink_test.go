package magiclink

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

type mockCodeStore struct {
	createEmail   string
	createCode    string
	createToken   string
	createExpires time.Time
	createErr     error

	consumeEmail string
	consumeCode  string
	consumeErr   error

	lookupEmail string
	lookupCode  string
	lookupErr   error
}

func (m *mockCodeStore) Create(_ context.Context, email, code, token string, expiresAt time.Time) error {
	m.createEmail = email
	m.createCode = code
	m.createToken = token
	m.createExpires = expiresAt
	return m.createErr
}

func (m *mockCodeStore) ConsumeByCode(_ context.Context, email, code string) error {
	m.consumeEmail = email
	m.consumeCode = code
	return m.consumeErr
}

func (m *mockCodeStore) LookupByToken(_ context.Context, _ string) (string, string, error) {
	return m.lookupEmail, m.lookupCode, m.lookupErr
}

type mockUserStore struct {
	upsertIdentity string
	upsertEmail    string
	upsertName     string
	upsertID       string
	upsertErr      error

	getEmail string
	getID    string
	getName  string
	getErr   error
}

func (m *mockUserStore) UpsertUser(_ context.Context, identityKey, email, displayName string) (string, error) {
	m.upsertIdentity = identityKey
	m.upsertEmail = email
	m.upsertName = displayName
	if m.upsertErr != nil {
		return "", m.upsertErr
	}
	if m.upsertID == "" {
		return "user-default", nil
	}
	return m.upsertID, nil
}

func (m *mockUserStore) GetUserByEmail(_ context.Context, email string) (string, string, error) {
	m.getEmail = email
	if m.getErr != nil {
		return "", "", m.getErr
	}
	return m.getID, m.getName, nil
}

type mockEmailSender struct {
	to      string
	subject string
	body    string
	err     error
}

func (m *mockEmailSender) Send(_ context.Context, to, subject, htmlBody string) error {
	m.to = to
	m.subject = subject
	m.body = htmlBody
	return m.err
}

type mockRenderer struct {
	code     string
	magicURL string
	appName  string
}

func (m *mockRenderer) Render(code, magicURL, appName string) (string, string) {
	m.code = code
	m.magicURL = magicURL
	m.appName = appName
	return "subject", "html"
}

func testConfig() Config {
	return Config{
		JWTSecret: strings.Repeat("x", 32),
		AppURL:    "https://api.example.com",
		AppName:   "Example",
	}
}

func TestSendStoresAndEmails(t *testing.T) {
	codes := &mockCodeStore{}
	users := &mockUserStore{}
	email := &mockEmailSender{}
	renderer := &mockRenderer{}

	svc := New(testConfig(), codes, users, email, renderer)
	now := time.Date(2026, 3, 26, 8, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	if err := svc.Send(context.Background(), "  User@Example.com "); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if codes.createEmail != "user@example.com" {
		t.Fatalf("create email = %q, want normalized", codes.createEmail)
	}
	if ok, _ := regexp.MatchString(`^\d{6}$`, codes.createCode); !ok {
		t.Fatalf("code %q is not 6 digits", codes.createCode)
	}
	if len(codes.createToken) != 64 {
		t.Fatalf("token length = %d, want 64", len(codes.createToken))
	}
	if !codes.createExpires.Equal(now.Add(DefaultCodeTTL)) {
		t.Fatalf("expiresAt = %v, want %v", codes.createExpires, now.Add(DefaultCodeTTL))
	}
	if renderer.appName != "Example" {
		t.Fatalf("renderer app name = %q", renderer.appName)
	}
	if !strings.HasPrefix(renderer.magicURL, "https://api.example.com/auth/verify?token=") {
		t.Fatalf("unexpected magic URL %q", renderer.magicURL)
	}
	if email.to != "user@example.com" || email.subject != "subject" || email.body != "html" {
		t.Fatalf("email payload mismatch: to=%q subject=%q body=%q", email.to, email.subject, email.body)
	}
}

func TestVerifyCodeReturnsJWT(t *testing.T) {
	codes := &mockCodeStore{}
	users := &mockUserStore{upsertID: "user-123", getID: "user-123", getName: "Brendan"}
	email := &mockEmailSender{}

	svc := New(testConfig(), codes, users, email, nil)
	svc.now = time.Now

	result, err := svc.VerifyCode(context.Background(), "Brendan@Example.com", "123456")
	if err != nil {
		t.Fatalf("VerifyCode() error = %v", err)
	}

	if result.UserID != "user-123" {
		t.Fatalf("user id = %q, want user-123", result.UserID)
	}
	if result.Email != "brendan@example.com" {
		t.Fatalf("email = %q", result.Email)
	}
	if result.DisplayName != "Brendan" {
		t.Fatalf("display name = %q", result.DisplayName)
	}
	if result.JWT == "" {
		t.Fatalf("jwt is empty")
	}

	claims, err := svc.ValidateJWT(result.JWT)
	if err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}
	if claims.Subject != "email|brendan@example.com" {
		t.Fatalf("subject = %q", claims.Subject)
	}
}

func TestVerifyTokenMapsCodeFailuresToInvalidToken(t *testing.T) {
	codes := &mockCodeStore{
		lookupEmail: "user@example.com",
		lookupCode:  "123456",
		consumeErr:  ErrExpiredCode,
	}
	users := &mockUserStore{}
	email := &mockEmailSender{}
	svc := New(testConfig(), codes, users, email, nil)

	_, err := svc.VerifyToken(context.Background(), "token")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("VerifyToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestAuthenticateBearerAndMiddleware(t *testing.T) {
	codes := &mockCodeStore{}
	users := &mockUserStore{upsertID: "user-42", getID: "user-42", getName: "Dev"}
	email := &mockEmailSender{}
	svc := New(testConfig(), codes, users, email, nil)

	token, err := svc.IssueToken(Claims{
		Subject:     "email|dev@example.com",
		Email:       "dev@example.com",
		DisplayName: "Dev",
	})
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	userID, claims, err := svc.AuthenticateBearer(context.Background(), "Bearer "+token)
	if err != nil {
		t.Fatalf("AuthenticateBearer() error = %v", err)
	}
	if userID != "user-42" {
		t.Fatalf("userID = %q", userID)
	}
	if claims.Email != "dev@example.com" {
		t.Fatalf("claims email = %q", claims.Email)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatalf("user id missing from context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(gotUserID))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	svc.Middleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("middleware status = %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "user-42" {
		t.Fatalf("middleware body = %q", rr.Body.String())
	}
}
