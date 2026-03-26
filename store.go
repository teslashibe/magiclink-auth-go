package magiclink

import (
	"context"
	"time"
)

// CodeStore persists and consumes one-time login codes.
//
// Implementations should return package sentinel errors where possible:
// ErrInvalidCode, ErrExpiredCode, ErrCodeAlreadyUsed, ErrInvalidToken, ErrExpiredToken, ErrTokenAlreadyUsed.
type CodeStore interface {
	Create(ctx context.Context, email, code, token string, expiresAt time.Time) error
	ConsumeByCode(ctx context.Context, email, code string) error
	LookupByToken(ctx context.Context, token string) (email, code string, err error)
}

// UserStore is owned by the consuming application and bridges identity into app users.
type UserStore interface {
	UpsertUser(ctx context.Context, identityKey, email, displayName string) (userID string, err error)
	GetUserByEmail(ctx context.Context, email string) (userID, displayName string, err error)
}
