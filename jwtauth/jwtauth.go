// Package jwtauth provides a minimal HMAC-SHA256 (HS256) JWT verifier and
// signer for Helixon fleet services. It is intentionally minimal: no JWKS
// endpoint, no asymmetric keys, no key rotation infrastructure — those
// belong in helixon-platform's auth service once we cross the GA threshold.
//
// Per v18680-1 (extract internal/auth to helixon-common) and v18680-2
// (sprintboard-mcp JWT middleware). Sprintboard-mcp and helixon-platform
// share the same secret via the HELIXON_JWT_SECRET environment variable
// (operator-gated; produced by helixon-platform's secrets-bootstrap).
package jwtauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidToken is returned when the token signature or format fails.
var ErrInvalidToken = errors.New("invalid token")

// ErrExpiredToken is returned when the token has expired.
var ErrExpiredToken = errors.New("expired token")

// ErrMissingClaim is returned when a required claim is absent from the token.
var ErrMissingClaim = errors.New("missing required claim")

// Claims is the canonical Helixon JWT claim set. TenantID is the multi-tenancy
// partition key (per v18675-3); Subject identifies the actor (machine, agent,
// or human); Scopes limits which operations the bearer can perform.
type Claims struct {
	Subject  string   `json:"sub"`
	TenantID string   `json:"tenant_id"`
	Scopes   []string `json:"scopes,omitempty"`
	Issuer   string   `json:"iss,omitempty"`
	Expires  int64    `json:"exp"`
	IssuedAt int64    `json:"iat,omitempty"`
}

// HasScope reports whether the bearer holds the named scope.
func (c Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Verifier validates HS256-signed JWTs against a shared secret.
type Verifier struct {
	secret []byte
	// Issuer, if non-empty, is enforced: tokens with iss != v.Issuer are
	// rejected with ErrInvalidToken. Default off (no issuer check).
	Issuer string
	// ClockSkew allows small NTP drift between signer and verifier.
	ClockSkew time.Duration
}

// NewVerifier returns a Verifier bound to secret. An empty secret is rejected.
func NewVerifier(secret []byte, issuer string) (*Verifier, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be at least 32 bytes (got %d)", len(secret))
	}
	return &Verifier{secret: secret, Issuer: issuer, ClockSkew: 30 * time.Second}, nil
}

// Verify parses a JWT string and returns its claims. Errors are typed:
// ErrInvalidToken for signature/format failures, ErrExpiredToken for expiry,
// ErrMissingClaim for missing required claims.
func (v *Verifier) Verify(tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 parts, got %d", ErrInvalidToken, len(parts))
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: header b64: %v", ErrInvalidToken, err)
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload b64: %v", ErrInvalidToken, err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: signature b64: %v", ErrInvalidToken, err)
	}

	// Parse header FIRST so alg=none is rejected before we trust the
	// signature at all (defence in depth against the classic JWT bypass).
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("%w: header json: %v", ErrInvalidToken, err)
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("%w: unsupported alg %q", ErrInvalidToken, header.Alg)
	}

	// Recompute HMAC over header.payload
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, sig) {
		return nil, fmt.Errorf("%w: signature mismatch", ErrInvalidToken)
	}

	// Parse claims
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("%w: payload json: %v", ErrInvalidToken, err)
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: sub", ErrMissingClaim)
	}
	if claims.TenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id", ErrMissingClaim)
	}
	if claims.Expires == 0 {
		return nil, fmt.Errorf("%w: exp", ErrMissingClaim)
	}

	// Expiry check (with clock skew tolerance)
	now := time.Now().Unix()
	if now > claims.Expires+int64(v.ClockSkew.Seconds()) {
		return nil, fmt.Errorf("%w (exp=%d, now=%d)", ErrExpiredToken, claims.Expires, now)
	}

	// Issuer check (if enforced)
	if v.Issuer != "" && claims.Issuer != v.Issuer {
		return nil, fmt.Errorf("%w: issuer %q != %q", ErrInvalidToken, claims.Issuer, v.Issuer)
	}

	return &claims, nil
}

// Sign produces a HS256-signed JWT for the given claims, valid for ttl.
// Used by helixon-platform's auth service and by tests; production callers
// should source claims from a request context, not from freeform input.
//
// A negative or zero ttl produces a token whose exp is in the past; this
// is intentional for tests that exercise the expiry path without time-warping.
// In production, callers must pass a positive ttl.
func (v *Verifier) Sign(claims Claims, ttl time.Duration) (string, error) {
	if len(claims.TenantID) == 0 {
		return "", fmt.Errorf("%w: tenant_id", ErrMissingClaim)
	}
	now := time.Now().Unix()
	claims.IssuedAt = now
	claims.Expires = now + int64(ttl.Seconds())
	if claims.Issuer == "" {
		claims.Issuer = v.Issuer
	}

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}
