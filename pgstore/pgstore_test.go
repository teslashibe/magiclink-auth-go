package pgstore

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/teslashibe/magiclink-auth-go"
)

func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("MAGICLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MAGICLINK_TEST_DATABASE_URL to run pgstore integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	defer pool.Close()

	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	store := New(pool)

	email := "integration@example.com"
	code := "123456"
	token := "integration-token"

	if err := store.Create(ctx, email, code, token, time.Now().Add(10*time.Minute)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	gotEmail, gotCode, err := store.LookupByToken(ctx, token)
	if err != nil {
		t.Fatalf("LookupByToken() error = %v", err)
	}
	if gotEmail != email || gotCode != code {
		t.Fatalf("LookupByToken() = (%q, %q)", gotEmail, gotCode)
	}

	if err := store.ConsumeByCode(ctx, email, code); err != nil {
		t.Fatalf("ConsumeByCode() error = %v", err)
	}
	if err := store.ConsumeByCode(ctx, email, code); !errors.Is(err, magiclink.ErrCodeAlreadyUsed) {
		t.Fatalf("second ConsumeByCode() error = %v, want ErrCodeAlreadyUsed", err)
	}

	userID, err := store.UpsertUser(ctx, "email|integration@example.com", email, "integration")
	if err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}
	if userID == "" {
		t.Fatalf("UpsertUser() returned empty user id")
	}

	gotID, gotName, err := store.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}
	if gotID == "" || gotName == "" {
		t.Fatalf("GetUserByEmail() returned empty values: id=%q name=%q", gotID, gotName)
	}
}
