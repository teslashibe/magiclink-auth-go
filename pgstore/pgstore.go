package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/teslashibe/magiclink-auth-go"
)

// Store is a PostgreSQL reference implementation of magiclink stores.
type Store struct {
	db *pgxpool.Pool
}

var (
	_ magiclink.CodeStore = (*Store)(nil)
	_ magiclink.UserStore = (*Store)(nil)
)

// New returns a PostgreSQL store.
func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) ensureDB() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("pgstore: nil database pool")
	}
	return nil
}

// Create stores a new one-time code row.
func (s *Store) Create(ctx context.Context, email, code, token string, expiresAt time.Time) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO auth_codes (email, code, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, strings.TrimSpace(strings.ToLower(email)), strings.TrimSpace(code), strings.TrimSpace(token), expiresAt)
	if err != nil {
		return fmt.Errorf("insert auth code: %w", err)
	}
	return nil
}

// ConsumeByCode atomically marks a code as used.
func (s *Store) ConsumeByCode(ctx context.Context, email, code string) error {
	if err := s.ensureDB(); err != nil {
		return err
	}

	email = strings.TrimSpace(strings.ToLower(email))
	code = strings.TrimSpace(code)

	tag, err := s.db.Exec(ctx, `
		UPDATE auth_codes
		SET used = true, used_at = NOW()
		WHERE id = (
			SELECT id
			FROM auth_codes
			WHERE email = $1 AND code = $2 AND used = false AND expires_at > NOW()
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, email, code)
	if err != nil {
		return fmt.Errorf("consume code: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	var used bool
	var expiresAt time.Time
	err = s.db.QueryRow(ctx, `
		SELECT used, expires_at
		FROM auth_codes
		WHERE email = $1 AND code = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, email, code).Scan(&used, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return magiclink.ErrInvalidCode
		}
		return fmt.Errorf("classify code failure: %w", err)
	}

	switch {
	case used:
		return magiclink.ErrCodeAlreadyUsed
	case !expiresAt.After(time.Now()):
		return magiclink.ErrExpiredCode
	default:
		return magiclink.ErrInvalidCode
	}
}

// LookupByToken resolves a token to email+code if still valid.
func (s *Store) LookupByToken(ctx context.Context, token string) (email, code string, err error) {
	if err := s.ensureDB(); err != nil {
		return "", "", err
	}
	token = strings.TrimSpace(token)

	var used bool
	var expiresAt time.Time
	err = s.db.QueryRow(ctx, `
		SELECT email, code, used, expires_at
		FROM auth_codes
		WHERE token = $1
		LIMIT 1
	`, token).Scan(&email, &code, &used, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", magiclink.ErrInvalidToken
		}
		return "", "", fmt.Errorf("lookup token: %w", err)
	}

	switch {
	case used:
		return "", "", magiclink.ErrTokenAlreadyUsed
	case !expiresAt.After(time.Now()):
		return "", "", magiclink.ErrExpiredToken
	default:
		return strings.TrimSpace(strings.ToLower(email)), strings.TrimSpace(code), nil
	}
}

// UpsertUser inserts or updates a user row and returns user id.
func (s *Store) UpsertUser(ctx context.Context, identityKey, email, displayName string) (string, error) {
	if err := s.ensureDB(); err != nil {
		return "", err
	}

	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (identity_key, email, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (identity_key)
		DO UPDATE SET
			email = EXCLUDED.email,
			display_name = EXCLUDED.display_name,
			updated_at = NOW()
		RETURNING id
	`, strings.TrimSpace(identityKey), strings.TrimSpace(strings.ToLower(email)), strings.TrimSpace(displayName)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert user: %w", err)
	}
	return id, nil
}

// GetUserByEmail finds the latest user row by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (string, string, error) {
	if err := s.ensureDB(); err != nil {
		return "", "", err
	}
	email = strings.TrimSpace(strings.ToLower(email))

	var id string
	var displayName string
	err := s.db.QueryRow(ctx, `
		SELECT id, COALESCE(display_name, '')
		FROM users
		WHERE email = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, email).Scan(&id, &displayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", fmt.Errorf("user not found")
		}
		return "", "", fmt.Errorf("get user by email: %w", err)
	}
	return id, displayName, nil
}
