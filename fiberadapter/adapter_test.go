package fiberadapter

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/teslashibe/magiclink-auth-go"
)

type testCodeStore struct {
	createErr   error
	consumeErr  error
	lookupEmail string
	lookupCode  string
	lookupErr   error
}

func (s *testCodeStore) Create(context.Context, string, string, string, time.Time) error {
	return s.createErr
}

func (s *testCodeStore) ConsumeByCode(context.Context, string, string) error {
	return s.consumeErr
}

func (s *testCodeStore) LookupByToken(context.Context, string) (string, string, error) {
	return s.lookupEmail, s.lookupCode, s.lookupErr
}

type testUserStore struct {
	id  string
	err error
}

func (s *testUserStore) UpsertUser(context.Context, string, string, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.id == "" {
		return "user-test", nil
	}
	return s.id, nil
}

func (s *testUserStore) GetUserByEmail(context.Context, string) (string, string, error) {
	return s.id, "Test", nil
}

type testEmailSender struct{}

func (testEmailSender) Send(context.Context, string, string, string) error { return nil }

func newService(codeStore *testCodeStore) *magiclink.Service {
	return magiclink.New(
		magiclink.Config{
			JWTSecret:   strings.Repeat("x", 32),
			AppURL:      "https://api.example.com",
			AppName:     "Example",
			DeepLinkURL: "example://auth",
		},
		codeStore,
		&testUserStore{id: "user-fiber"},
		testEmailSender{},
		nil,
	)
}

func TestVerifyCodeHandlerUnauthorized(t *testing.T) {
	app := fiber.New()
	svc := newService(&testCodeStore{consumeErr: magiclink.ErrInvalidCode})
	app.Post("/auth/verify", VerifyCodeHandler(svc))

	req := httptest.NewRequest("POST", "/auth/verify", strings.NewReader(`{"email":"a@b.com","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestVerifyLinkHandlerReturnsHTML(t *testing.T) {
	app := fiber.New()
	svc := newService(&testCodeStore{
		lookupEmail: "a@b.com",
		lookupCode:  "123456",
	})
	app.Get("/auth/verify", VerifyLinkHandler(svc))

	req := httptest.NewRequest("GET", "/auth/verify?token=abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content type = %q, want text/html", got)
	}
}

func TestAuthMiddlewareSetsUserID(t *testing.T) {
	app := fiber.New()
	svc := newService(&testCodeStore{})

	token, err := svc.IssueToken(magiclink.Claims{
		Subject:     "email|test@example.com",
		Email:       "test@example.com",
		DisplayName: "Test",
	})
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	app.Use(AuthMiddleware(svc))
	app.Get("/secure", func(c *fiber.Ctx) error {
		uid, _ := c.Locals("user_id").(string)
		return c.SendString(uid)
	})

	req := httptest.NewRequest("GET", "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}
