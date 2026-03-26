# magiclink-auth-go

Production-grade passwordless email auth for Go:

- 6-digit OTP code + clickable magic link
- Single-use code consumption
- HS256 JWT issuance and validation
- `net/http` handlers + middleware
- Fiber adapter
- PostgreSQL reference store
- Resend email sender

## Install

```bash
go get github.com/teslashibe/magiclink-auth-go
```

## Package Layout

- `magiclink` (root): core service, net/http handlers, middleware
- `pgstore`: PostgreSQL `CodeStore` + `UserStore`, embedded migrations
- `resend`: Resend `EmailSender`
- `fiberadapter`: Fiber handlers + middleware

## Quickstart (net/http)

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/teslashibe/magiclink-auth-go"
	"github.com/teslashibe/magiclink-auth-go/pgstore"
	"github.com/teslashibe/magiclink-auth-go/resend"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	if err := pgstore.ApplyMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	store := pgstore.New(pool)
	svc := magiclink.New(
		magiclink.Config{
			JWTSecret:   os.Getenv("JWT_SECRET"),
			AppURL:      "https://api.myapp.com",
			AppName:     "MyApp",
			FromAddress: "MyApp <noreply@myapp.com>",
			DeepLinkURL: "myapp://auth", // optional
		},
		store,
		store,
		resend.New(os.Getenv("RESEND_API_KEY"), "MyApp <noreply@myapp.com>"),
		nil, // nil => DefaultEmailRenderer
	)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/magic-link", svc.HandleSend)
	mux.HandleFunc("POST /auth/verify", svc.HandleVerifyCode)
	mux.HandleFunc("GET /auth/verify", svc.HandleVerifyLink)

	protected := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := magiclink.UserIDFromContext(r.Context())
		w.Write([]byte(userID))
	}))
	mux.Handle("GET /api/me", protected)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Quickstart (Fiber)

```go
store := pgstore.New(pool)
svc := magiclink.New(
	magiclink.Config{
		JWTSecret:   os.Getenv("JWT_SECRET"),
		AppURL:      "https://api.myapp.com",
		AppName:     "MyApp",
		FromAddress: "MyApp <noreply@myapp.com>",
		DeepLinkURL: "myapp://auth",
	},
	store,
	store,
	resend.New(os.Getenv("RESEND_API_KEY"), "MyApp <noreply@myapp.com>"),
	nil,
)

app.Post("/auth/magic-link", fiberadapter.SendHandler(svc))
app.Post("/auth/verify", fiberadapter.VerifyCodeHandler(svc))
app.Get("/auth/verify", fiberadapter.VerifyLinkHandler(svc))

api := app.Group("/api", fiberadapter.AuthMiddleware(svc))
api.Get("/me", func(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"user_id": c.Locals("user_id")})
})
```

## Core Interfaces

Implement these to customize behavior:

- `magiclink.CodeStore`
- `magiclink.UserStore`
- `magiclink.EmailSender`
- `magiclink.EmailRenderer`

This keeps the core service framework- and provider-agnostic.

## HTTP API

- `POST /auth/magic-link` with `{"email":"user@example.com"}`
- `POST /auth/verify` with `{"email":"user@example.com","code":"123456"}`
- `GET /auth/verify?token=...`

`POST /auth/verify` success response:

```json
{
  "token": "jwt",
  "user_id": "uuid",
  "email": "user@example.com",
  "name": "user"
}
```

## Configuration

```go
type Config struct {
	JWTSecret   string        // required, min 32 chars
	AppURL      string        // required, absolute URL
	AppName     string        // default "App"
	FromAddress string        // optional, useful for sender config
	CodeTTL     time.Duration // default 10m
	TokenTTL    time.Duration // default 30d
	DeepLinkURL string        // optional: myapp://auth
	CodeLength  int           // default 6 (range 4..10)
}
```

## Security Notes

- Uses `crypto/rand` for code + token generation.
- Codes are one-time use via `ConsumeByCode`.
- JWT algorithm is pinned to HS256.
- `Authorization` parsing is strict (`Bearer <token>`).
- Prefer a high-entropy `JWT_SECRET` (32+ chars recommended).

## Testing

Run unit tests:

```bash
go test ./...
```

Run pgstore integration test (requires reachable Postgres):

```bash
MAGICLINK_TEST_DATABASE_URL=postgres://... go test ./pgstore -run Integration
```
