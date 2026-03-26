package magiclink

import (
	"errors"
	"net/http"
)

var (
	// Request validation errors.
	ErrMissingEmail = errors.New("email required")
	ErrMissingCode  = errors.New("code required")
	ErrMissingToken = errors.New("token required")

	// Code/token lifecycle errors.
	ErrInvalidCode      = errors.New("invalid code")
	ErrExpiredCode      = errors.New("expired code")
	ErrCodeAlreadyUsed  = errors.New("code already used")
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("expired token")
	ErrTokenAlreadyUsed = errors.New("token already used")

	// Authorization/JWT errors.
	ErrMissingAuthorization = errors.New("missing authorization header")
	ErrInvalidAuthorization = errors.New("invalid authorization format")
	ErrInvalidJWT           = errors.New("invalid token")
	ErrExpiredJWT           = errors.New("expired token")

	// Initialization/configuration errors.
	ErrInvalidConfig  = errors.New("invalid config")
	ErrNotInitialized = errors.New("service not initialized")
)

// HTTPStatus maps known package errors to HTTP status codes.
func HTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrMissingEmail),
		errors.Is(err, ErrMissingCode),
		errors.Is(err, ErrMissingToken):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidCode),
		errors.Is(err, ErrExpiredCode),
		errors.Is(err, ErrCodeAlreadyUsed),
		errors.Is(err, ErrInvalidToken),
		errors.Is(err, ErrExpiredToken),
		errors.Is(err, ErrTokenAlreadyUsed),
		errors.Is(err, ErrMissingAuthorization),
		errors.Is(err, ErrInvalidAuthorization),
		errors.Is(err, ErrInvalidJWT),
		errors.Is(err, ErrExpiredJWT):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// PublicError returns a safe client-facing error message.
func PublicError(err error) string {
	switch {
	case errors.Is(err, ErrMissingEmail):
		return ErrMissingEmail.Error()
	case errors.Is(err, ErrMissingCode):
		return ErrMissingCode.Error()
	case errors.Is(err, ErrMissingToken):
		return ErrMissingToken.Error()
	case errors.Is(err, ErrInvalidCode):
		return "invalid or expired code"
	case errors.Is(err, ErrExpiredCode):
		return "invalid or expired code"
	case errors.Is(err, ErrCodeAlreadyUsed):
		return "invalid or expired code"
	case errors.Is(err, ErrInvalidToken):
		return "invalid or expired link"
	case errors.Is(err, ErrExpiredToken):
		return "invalid or expired link"
	case errors.Is(err, ErrTokenAlreadyUsed):
		return "invalid or expired link"
	case errors.Is(err, ErrMissingAuthorization):
		return ErrMissingAuthorization.Error()
	case errors.Is(err, ErrInvalidAuthorization):
		return ErrInvalidAuthorization.Error()
	case errors.Is(err, ErrInvalidJWT):
		return ErrInvalidJWT.Error()
	case errors.Is(err, ErrExpiredJWT):
		return ErrExpiredJWT.Error()
	case errors.Is(err, ErrInvalidConfig), errors.Is(err, ErrNotInitialized):
		return "server misconfigured"
	default:
		return "internal server error"
	}
}
