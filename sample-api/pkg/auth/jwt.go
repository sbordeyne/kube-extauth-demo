// Package auth signs the JWTs handed out by the login endpoint.
//
// Note: this service does NOT verify inbound tokens — authorization is done by
// OPA at the gateway (ext-auth). Tokens reaching handlers are assumed trusted.
package auth

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "sample-api"

// LoadPrivateKey reads a PEM-encoded RSA private key (PKCS#1 or PKCS#8).
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", path, err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		return nil, fmt.Errorf("parse private key %q: %w", path, err)
	}
	return key, nil
}

// Signer mints RS256 tokens.
type Signer struct {
	key *rsa.PrivateKey
}

// NewSigner builds a Signer from an RSA private key.
func NewSigner(key *rsa.PrivateKey) *Signer {
	return &Signer{key: key}
}

// Sign issues a token for a user, embedding their role and scopes.
func (s *Signer) Sign(sub uuid.UUID, role string, scopes []string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(ttl)
	jti, err := uuid.NewV7()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate jti: %w", err)
	}
	claims := jwt.MapClaims{
		"iss":    issuer,
		"sub":    sub.String(),
		"role":   role,
		"scopes": scopes,
		"iat":    now.Unix(),
		"exp":    expiresAt.Unix(),
		"jti":    jti.String(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token, err = t.SignedString(s.key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, expiresAt, nil
}
