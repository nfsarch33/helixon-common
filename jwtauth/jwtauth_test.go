package jwtauth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func testSecret() []byte {
	return []byte("0123456789abcdef0123456789abcdef0123456789abcdef") // 54 bytes
}

func TestNewVerifier_RejectsShortSecret(t *testing.T) {
	if _, err := NewVerifier([]byte("short"), ""); err == nil {
		t.Fatal("expected error for short secret")
	}
	if _, err := NewVerifier(testSecret(), ""); err != nil {
		t.Fatalf("valid-length secret rejected: %v", err)
	}
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	v, err := NewVerifier(testSecret(), "helixon-test")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	tok, err := v.Sign(Claims{
		Subject:  "agent:helixon-platform",
		TenantID: "tenant-abc",
		Scopes:   []string{"read", "write"},
	}, 1*time.Hour)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if strings.Count(tok, ".") != 2 {
		t.Errorf("token must have 2 dots, got %d in %s", strings.Count(tok, "."), tok)
	}
	claims, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "agent:helixon-platform" {
		t.Errorf("Subject = %q, want agent:helixon-platform", claims.Subject)
	}
	if claims.TenantID != "tenant-abc" {
		t.Errorf("TenantID = %q, want tenant-abc", claims.TenantID)
	}
	if !claims.HasScope("read") || !claims.HasScope("write") {
		t.Errorf("scopes mismatch: %v", claims.Scopes)
	}
	if claims.HasScope("admin") {
		t.Errorf("HasScope(admin) should be false")
	}
}

func TestVerify_RejectsBadSignature(t *testing.T) {
	v, err := NewVerifier(testSecret(), "")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	tok, err := v.Sign(Claims{Subject: "x", TenantID: "t"}, 1*time.Hour)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	// Tamper with signature
	tampered := tok[:len(tok)-2] + "XX"
	_, err = v.Verify(tampered)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerify_RejectsExpired(t *testing.T) {
	v, err := NewVerifier(testSecret(), "")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	v.ClockSkew = 0
	// Sign with a -60s TTL so the resulting exp is already in the past.
	tok, err := v.Sign(Claims{Subject: "x", TenantID: "t"}, -60*time.Second)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_, err = v.Verify(tok)
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestVerify_RejectsMissingSubject(t *testing.T) {
	v, err := NewVerifier(testSecret(), "")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	// Sign with no sub — Sign permits empty sub so Verify can reject it
	// (so tests don't have to construct invalid tokens by hand).
	tok, err := v.Sign(Claims{TenantID: "t"}, 1*time.Hour)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_, err = v.Verify(tok)
	if !errors.Is(err, ErrMissingClaim) {
		t.Errorf("expected ErrMissingClaim, got %v", err)
	}
}

func TestVerify_IssuerEnforced(t *testing.T) {
	v, err := NewVerifier(testSecret(), "helixon-prod")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	// Sign with different issuer
	tok, err := v.Sign(Claims{
		Subject:  "x",
		TenantID: "t",
		Issuer:   "helixon-test",
	}, 1*time.Hour)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_, err = v.Verify(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken on issuer mismatch, got %v", err)
	}
}

func TestVerify_RejectsAlgNone(t *testing.T) {
	// alg=none is the classic JWT bypass attack. We must reject.
	// Construct a hand-rolled alg=none token.
	noneTok := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ4IiwidGVuYW50X2lkIjoidCJ9."
	_, err := (&Verifier{secret: testSecret()}).Verify(noneTok)
	if err == nil {
		t.Fatal("alg=none token must be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported alg") {
		t.Errorf("expected unsupported-alg error, got %v", err)
	}
}

func TestVerify_RejectsMalformedTokens(t *testing.T) {
	v, err := NewVerifier(testSecret(), "")
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	cases := []string{"", "abc", "abc.def", "abc.def.ghi.jkl", "not.a.jwt"}
	for _, c := range cases {
		if _, err := v.Verify(c); err == nil {
			t.Errorf("expected error for malformed token %q", c)
		}
	}
}
