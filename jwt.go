package magiclink

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the package's application-level JWT claims.
type Claims struct {
	Subject     string
	Email       string
	DisplayName string
}

type appJWTClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.RegisteredClaims
}

func issueToken(secret string, ttl time.Duration, now time.Time, claims Claims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("%w: missing jwt secret", ErrInvalidConfig)
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return "", fmt.Errorf("%w: missing subject", ErrInvalidJWT)
	}

	j := appJWTClaims{
		Email: claims.Email,
		Name:  claims.DisplayName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.Subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, j)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tokenStr, nil
}

func validateToken(secret, tokenStr string) (*Claims, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return nil, ErrInvalidJWT
	}

	claims := &appJWTClaims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrExpiredJWT
		default:
			return nil, ErrInvalidJWT
		}
	}
	if !token.Valid {
		return nil, ErrInvalidJWT
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, ErrInvalidJWT
	}

	return &Claims{
		Subject:     claims.Subject,
		Email:       claims.Email,
		DisplayName: claims.Name,
	}, nil
}
