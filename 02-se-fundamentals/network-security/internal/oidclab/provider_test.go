package oidclab

import (
	"testing"
	"time"
)

func TestAuthorizationCodePKCEAndClaims(t *testing.T) {
	t.Parallel()
	provider, err := NewProvider("http://127.0.0.1/issuer")
	if err != nil {
		t.Fatal(err)
	}
	provider.RegisterClient("rp-a", "http://127.0.0.1/callback")
	verifier := "test-verifier-with-enough-entropy"
	code, err := provider.Authorize(
		"rp-a", "http://127.0.0.1/callback", "subject-1", "nonce-1", PKCEChallenge(verifier),
	)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := provider.Redeem(code, "rp-a", "http://127.0.0.1/callback", verifier)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := provider.ValidateIDToken(tokens.IDToken, "http://127.0.0.1/issuer", "rp-a", "nonce-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "subject-1" {
		t.Fatalf("subject=%q", claims.Subject)
	}
	if _, err := provider.Redeem(code, "rp-a", "http://127.0.0.1/callback", verifier); err == nil {
		t.Fatal("authorization code replay succeeded")
	}
}

func TestNegativeClaimsAndRedirects(t *testing.T) {
	t.Parallel()
	provider, err := NewProvider("http://127.0.0.1/issuer")
	if err != nil {
		t.Fatal(err)
	}
	provider.RegisterClient("rp-a", "http://127.0.0.1/callback")
	if _, err := provider.Authorize("rp-a", "http://127.0.0.1/other", "subject", "nonce", PKCEChallenge("v")); err == nil {
		t.Fatal("unregistered redirect URI succeeded")
	}
	if _, err := provider.Authorize(
		"rp-a",
		"http://127.0.0.1/callback?next=https%3A%2F%2Fattacker.invalid",
		"subject",
		"nonce",
		PKCEChallenge("v"),
	); err == nil {
		t.Fatal("open redirect-shaped redirect URI succeeded")
	}
	if _, err := provider.Redeem(
		"injected-authorization-code", "rp-a", "http://127.0.0.1/callback", "attacker-verifier",
	); err == nil {
		t.Fatal("authorization code injection succeeded")
	}
	now := time.Now()
	claims := Claims{
		Issuer: "http://127.0.0.1/issuer", Subject: "subject", Audience: "rp-a", Azp: "rp-a",
		Nonce: "nonce", Expires: now.Add(time.Minute).Unix(), IssuedAt: now.Unix(),
		NotBefore: now.Add(-time.Second).Unix(), SessionID: "sid-1",
	}
	token, err := provider.IssueTestToken(claims, "")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		issuer   string
		audience string
		nonce    string
		now      time.Time
	}{
		{"issuer", "wrong", "rp-a", "nonce", now},
		{"mix-up", "http://127.0.0.1/other-issuer", "rp-a", "nonce", now},
		{"audience", claims.Issuer, "rp-b", "nonce", now},
		{"token-audience-confusion", claims.Issuer, "resource-server-b", "nonce", now},
		{"nonce", claims.Issuer, "rp-a", "wrong", now},
		{"expiry", claims.Issuer, "rp-a", "nonce", now.Add(2 * time.Minute)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := provider.ValidateIDToken(token, test.issuer, test.audience, test.nonce, test.now); err == nil {
				t.Fatalf("%s mismatch unexpectedly succeeded", test.name)
			}
		})
	}
}
