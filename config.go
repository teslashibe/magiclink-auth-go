package magiclink

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultCodeTTL is the default amount of time a code is valid.
	DefaultCodeTTL = 10 * time.Minute
	// DefaultTokenTTL is the default JWT lifetime.
	DefaultTokenTTL = 30 * 24 * time.Hour
	// DefaultCodeLength is the default number of digits in OTP codes.
	DefaultCodeLength = 6

	minCodeLength   = 4
	maxCodeLength   = 10
	minJWTSecretLen = 32
)

// Config controls magic-link behavior.
type Config struct {
	// JWTSecret signs and validates HS256 JWTs.
	JWTSecret string
	// AppURL is the base API URL used to build verify links (e.g. https://api.example.com).
	AppURL string
	// AppName is displayed in default templates and token metadata.
	AppName string
	// FromAddress is a convenient place to store a default sender identity.
	FromAddress string
	// CodeTTL controls OTP code expiration.
	CodeTTL time.Duration
	// TokenTTL controls app JWT expiration.
	TokenTTL time.Duration
	// DeepLinkURL is optional (e.g. myapp://auth). When set, verify-link pages auto-redirect there.
	DeepLinkURL string
	// CodeLength controls OTP digits.
	CodeLength int
}

func (c Config) withDefaults() Config {
	if c.CodeTTL <= 0 {
		c.CodeTTL = DefaultCodeTTL
	}
	if c.TokenTTL <= 0 {
		c.TokenTTL = DefaultTokenTTL
	}
	if c.CodeLength <= 0 {
		c.CodeLength = DefaultCodeLength
	}
	if strings.TrimSpace(c.AppName) == "" {
		c.AppName = "App"
	}
	return c
}

// Validate validates config values.
func (c Config) Validate() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("jwt secret is required")
	}
	if len(strings.TrimSpace(c.JWTSecret)) < minJWTSecretLen {
		return fmt.Errorf("jwt secret must be at least %d characters", minJWTSecretLen)
	}

	appURL := strings.TrimSpace(c.AppURL)
	if appURL == "" {
		return fmt.Errorf("app url is required")
	}
	parsedAppURL, err := url.Parse(appURL)
	if err != nil || parsedAppURL.Scheme == "" || parsedAppURL.Host == "" {
		return fmt.Errorf("app url must be an absolute URL")
	}

	if strings.TrimSpace(c.FromAddress) != "" {
		if _, err := mail.ParseAddress(c.FromAddress); err != nil {
			return fmt.Errorf("from address is invalid: %w", err)
		}
	}

	if c.CodeTTL <= 0 {
		return fmt.Errorf("code ttl must be > 0")
	}
	if c.TokenTTL <= 0 {
		return fmt.Errorf("token ttl must be > 0")
	}
	if c.CodeLength < minCodeLength || c.CodeLength > maxCodeLength {
		return fmt.Errorf("code length must be between %d and %d", minCodeLength, maxCodeLength)
	}

	if strings.TrimSpace(c.DeepLinkURL) != "" {
		deep, err := url.Parse(c.DeepLinkURL)
		if err != nil {
			return fmt.Errorf("deep link url is invalid: %w", err)
		}
		if deep.Scheme == "" {
			return fmt.Errorf("deep link url must include a scheme")
		}
	}

	return nil
}
