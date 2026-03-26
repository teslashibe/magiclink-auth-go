package magiclink

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const emailIdentityPrefix = "email|"

// AuthResult is returned after successful verification.
type AuthResult struct {
	JWT         string `json:"token"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"name"`
}

// Service implements magic-link auth.
type Service struct {
	cfg    Config
	codes  CodeStore
	users  UserStore
	email  EmailSender
	render EmailRenderer

	now     func() time.Time
	initErr error
}

// New creates a new Service.
func New(cfg Config, codes CodeStore, users UserStore, email EmailSender, renderer EmailRenderer) *Service {
	cfg = cfg.withDefaults()
	if renderer == nil {
		renderer = DefaultEmailRenderer{}
	}

	svc := &Service{
		cfg:    cfg,
		codes:  codes,
		users:  users,
		email:  email,
		render: renderer,
		now:    time.Now,
	}

	var errs []error
	if err := cfg.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("%w: %v", ErrInvalidConfig, err))
	}
	if codes == nil {
		errs = append(errs, fmt.Errorf("%w: code store is nil", ErrNotInitialized))
	}
	if users == nil {
		errs = append(errs, fmt.Errorf("%w: user store is nil", ErrNotInitialized))
	}
	if email == nil {
		errs = append(errs, fmt.Errorf("%w: email sender is nil", ErrNotInitialized))
	}
	if len(errs) > 0 {
		svc.initErr = errors.Join(errs...)
	}

	return svc
}

func (s *Service) checkReady() error {
	if s == nil {
		return ErrNotInitialized
	}
	if s.initErr != nil {
		return s.initErr
	}
	return nil
}

// Send creates and emails a one-time code plus magic link.
func (s *Service) Send(ctx context.Context, email string) error {
	if err := s.checkReady(); err != nil {
		return err
	}

	email = normalizeEmail(email)
	if email == "" {
		return ErrMissingEmail
	}

	code, err := generateCode(s.cfg.CodeLength)
	if err != nil {
		return err
	}
	token, err := generateToken(linkTokenBytes)
	if err != nil {
		return err
	}

	expiresAt := s.now().Add(s.cfg.CodeTTL)
	if err := s.codes.Create(ctx, email, code, token, expiresAt); err != nil {
		return fmt.Errorf("create auth code: %w", err)
	}

	magicURL := strings.TrimRight(s.cfg.AppURL, "/") + "/auth/verify?token=" + url.QueryEscape(token)
	subject, htmlBody := s.render.Render(code, magicURL, s.cfg.AppName)

	if err := s.email.Send(ctx, email, subject, htmlBody); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// VerifyCode verifies a one-time code and returns a JWT plus user info.
func (s *Service) VerifyCode(ctx context.Context, email, code string) (*AuthResult, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	email = normalizeEmail(email)
	code = strings.TrimSpace(code)
	if email == "" {
		return nil, ErrMissingEmail
	}
	if code == "" {
		return nil, ErrMissingCode
	}

	if err := s.codes.ConsumeByCode(ctx, email, code); err != nil {
		return nil, err
	}

	return s.authenticateEmail(ctx, email)
}

// VerifyToken verifies a magic-link token and returns a JWT plus user info.
func (s *Service) VerifyToken(ctx context.Context, token string) (*AuthResult, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrMissingToken
	}

	email, code, err := s.codes.LookupByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	result, err := s.VerifyCode(ctx, email, code)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCode), errors.Is(err, ErrExpiredCode), errors.Is(err, ErrCodeAlreadyUsed):
			return nil, ErrInvalidToken
		default:
			return nil, err
		}
	}
	return result, nil
}

// VerifyTokenPage verifies a token and returns a success HTML page.
func (s *Service) VerifyTokenPage(ctx context.Context, token string) (string, error) {
	result, err := s.VerifyToken(ctx, token)
	if err != nil {
		return "", err
	}
	return s.SuccessPageHTML(result)
}

// SuccessPageHTML renders the default success page, including deep-link redirect when configured.
func (s *Service) SuccessPageHTML(result *AuthResult) (string, error) {
	if err := s.checkReady(); err != nil {
		return "", err
	}
	if result == nil {
		return "", fmt.Errorf("auth result is nil")
	}
	return renderVerifySuccessPage(s.cfg.AppName, s.cfg.DeepLinkURL, result)
}

// ValidateJWT parses and validates a JWT.
func (s *Service) ValidateJWT(tokenStr string) (*Claims, error) {
	if err := s.checkReady(); err != nil {
		return nil, err
	}
	return validateToken(s.cfg.JWTSecret, tokenStr)
}

// IssueToken issues a JWT for claims using the configured TTL.
func (s *Service) IssueToken(claims Claims) (string, error) {
	if err := s.checkReady(); err != nil {
		return "", err
	}
	return issueToken(s.cfg.JWTSecret, s.cfg.TokenTTL, s.now(), claims)
}

// AuthenticateBearer validates an Authorization header and upserts the user.
func (s *Service) AuthenticateBearer(ctx context.Context, authorizationHeader string) (userID string, claims *Claims, err error) {
	if err := s.checkReady(); err != nil {
		return "", nil, err
	}

	token, err := bearerToken(authorizationHeader)
	if err != nil {
		return "", nil, err
	}

	claims, err = s.ValidateJWT(token)
	if err != nil {
		return "", nil, err
	}

	userID, err = s.users.UpsertUser(ctx, claims.Subject, claims.Email, claims.DisplayName)
	if err != nil {
		return "", nil, fmt.Errorf("upsert user: %w", err)
	}
	if strings.TrimSpace(userID) == "" {
		return "", nil, fmt.Errorf("upsert user: empty user id")
	}

	return userID, claims, nil
}

func (s *Service) authenticateEmail(ctx context.Context, email string) (*AuthResult, error) {
	sub := emailIdentityPrefix + email
	displayName := emailToDisplayName(email)

	userID, err := s.users.UpsertUser(ctx, sub, email, displayName)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	if _, existingName, getErr := s.users.GetUserByEmail(ctx, email); getErr == nil && strings.TrimSpace(existingName) != "" {
		displayName = existingName
	}

	token, err := s.IssueToken(Claims{
		Subject:     sub,
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		JWT:         token,
		UserID:      userID,
		Email:       email,
		DisplayName: displayName,
	}, nil
}

func bearerToken(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", ErrMissingAuthorization
	}

	if len(header) < 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return "", ErrInvalidAuthorization
	}

	token := strings.TrimSpace(header[7:])
	if token == "" {
		return "", ErrInvalidAuthorization
	}
	return token, nil
}
