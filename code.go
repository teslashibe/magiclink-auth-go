package magiclink

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

const linkTokenBytes = 32

func generateCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("code length must be positive")
	}

	// 10^length
	max := big.NewInt(1)
	ten := big.NewInt(10)
	for i := 0; i < length; i++ {
		max.Mul(max, ten)
	}

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}

	raw := n.Text(10)
	if len(raw) >= length {
		return raw, nil
	}

	return strings.Repeat("0", length-len(raw)) + raw, nil
}

func generateToken(numBytes int) (string, error) {
	if numBytes <= 0 {
		return "", fmt.Errorf("token bytes must be positive")
	}
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func emailToDisplayName(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return email
	}
	return email[:at]
}
