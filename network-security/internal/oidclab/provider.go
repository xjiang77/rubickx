package oidclab

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

var rawURL = base64.RawURLEncoding

type Code struct {
	ClientID      string
	RedirectURI   string
	Subject       string
	Nonce         string
	CodeChallenge string
	ExpiresAt     time.Time
	Used          bool
}

type Claims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	Azp       string `json:"azp"`
	Nonce     string `json:"nonce"`
	Expires   int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
	SessionID string `json:"sid"`
}

type TokenSet struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKSet struct {
	Keys []JWK `json:"keys"`
}

type Provider struct {
	mu          sync.Mutex
	issuer      string
	clients     map[string]string
	codes       map[string]*Code
	keys        map[string]*rsa.PrivateKey
	activeKid   string
	retiredKids map[string]time.Time
	clock       func() time.Time
}

func NewProvider(issuer string) (*Provider, error) {
	p := &Provider{
		issuer:      issuer,
		clients:     make(map[string]string),
		codes:       make(map[string]*Code),
		keys:        make(map[string]*rsa.PrivateKey),
		retiredKids: make(map[string]time.Time),
		clock:       time.Now,
	}
	if err := p.rotateLocked("kid-1"); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) Issuer() string { return p.issuer }

func (p *Provider) RegisterClient(clientID, redirectURI string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[clientID] = redirectURI
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return rawURL.EncodeToString(buffer), nil
}

func PKCEChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return rawURL.EncodeToString(digest[:])
}

func (p *Provider) Authorize(clientID, redirectURI, subject, nonce, challenge string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	registered, ok := p.clients[clientID]
	if !ok || registered != redirectURI {
		return "", errors.New("redirect_uri is not exactly registered for client")
	}
	if nonce == "" || challenge == "" {
		return "", errors.New("nonce and PKCE challenge are required")
	}
	code, err := randomToken(24)
	if err != nil {
		return "", err
	}
	p.codes[code] = &Code{
		ClientID: clientID, RedirectURI: redirectURI, Subject: subject,
		Nonce: nonce, CodeChallenge: challenge, ExpiresAt: p.clock().Add(time.Minute),
	}
	return code, nil
}

func (p *Provider) Redeem(code, clientID, redirectURI, verifier string) (TokenSet, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	grant, ok := p.codes[code]
	if !ok || grant.Used {
		return TokenSet{}, errors.New("authorization code unknown or already used")
	}
	if p.clock().After(grant.ExpiresAt) {
		return TokenSet{}, errors.New("authorization code expired")
	}
	if grant.ClientID != clientID || grant.RedirectURI != redirectURI {
		return TokenSet{}, errors.New("authorization code binding mismatch")
	}
	if PKCEChallenge(verifier) != grant.CodeChallenge {
		return TokenSet{}, errors.New("PKCE verification failed")
	}
	grant.Used = true
	claims := Claims{
		Issuer: p.issuer, Subject: grant.Subject, Audience: clientID, Azp: clientID,
		Nonce: grant.Nonce, Expires: p.clock().Add(5 * time.Minute).Unix(),
		IssuedAt: p.clock().Unix(), NotBefore: p.clock().Add(-time.Second).Unix(),
		SessionID: "sid-" + code[:12],
	}
	token, err := p.signLocked(claims, p.activeKid)
	if err != nil {
		return TokenSet{}, err
	}
	access, err := randomToken(24)
	if err != nil {
		return TokenSet{}, err
	}
	return TokenSet{IDToken: token, AccessToken: access, TokenType: "Bearer", ExpiresIn: 300}, nil
}

func (p *Provider) IssueTestToken(claims Claims, kid string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if kid == "" {
		kid = p.activeKid
	}
	return p.signLocked(claims, kid)
}

func (p *Provider) Rotate(kid string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.activeKid != "" {
		p.retiredKids[p.activeKid] = p.clock().Add(2 * time.Minute)
	}
	return p.rotateLocked(kid)
}

func (p *Provider) rotateLocked(kid string) error {
	if _, exists := p.keys[kid]; exists {
		return fmt.Errorf("kid %s already exists", kid)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	p.keys[kid] = key
	p.activeKid = kid
	return nil
}

func (p *Provider) signLocked(claims Claims, kid string) (string, error) {
	key, ok := p.keys[kid]
	if !ok {
		return "", fmt.Errorf("unknown signing kid %s", kid)
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := rawURL.EncodeToString(header) + "." + rawURL.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + rawURL.EncodeToString(signature), nil
}

func (p *Provider) JWKSet() JWKSet {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.clock()
	keys := make([]JWK, 0, len(p.keys))
	for kid, key := range p.keys {
		if kid != p.activeKid {
			until, ok := p.retiredKids[kid]
			if !ok || now.After(until) {
				continue
			}
		}
		e := big.NewInt(int64(key.PublicKey.E)).Bytes()
		keys = append(keys, JWK{
			Kty: "RSA", Use: "sig", Alg: "RS256", Kid: kid,
			N: rawURL.EncodeToString(key.PublicKey.N.Bytes()), E: rawURL.EncodeToString(e),
		})
	}
	return JWKSet{Keys: keys}
}

func (p *Provider) ValidateIDToken(token, expectedIssuer, audience, nonce string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("token must contain three JWT parts")
	}
	headerBytes, err := rawURL.DecodeString(parts[0])
	if err != nil {
		return Claims{}, errors.New("invalid JWT header encoding")
	}
	var header map[string]string
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Claims{}, errors.New("invalid JWT header JSON")
	}
	if header["alg"] != "RS256" {
		return Claims{}, errors.New("unexpected JWT alg")
	}
	p.mu.Lock()
	key, ok := p.keys[header["kid"]]
	active := header["kid"] == p.activeKid
	retiredUntil := p.retiredKids[header["kid"]]
	p.mu.Unlock()
	if !ok || (!active && now.After(retiredUntil)) {
		return Claims{}, errors.New("unknown or expired JWT kid")
	}
	signature, err := rawURL.DecodeString(parts[2])
	if err != nil {
		return Claims{}, errors.New("invalid JWT signature encoding")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature); err != nil {
		return Claims{}, errors.New("JWT signature validation failed")
	}
	payload, err := rawURL.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid JWT payload encoding")
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, errors.New("invalid JWT claims")
	}
	if claims.Issuer != expectedIssuer {
		return Claims{}, errors.New("issuer mismatch")
	}
	if claims.Audience != audience || claims.Azp != audience {
		return Claims{}, errors.New("audience or authorized-party mismatch")
	}
	if claims.Nonce != nonce {
		return Claims{}, errors.New("nonce mismatch")
	}
	if now.Unix() >= claims.Expires {
		return Claims{}, errors.New("token expired")
	}
	if now.Unix() < claims.NotBefore {
		return Claims{}, errors.New("token not active")
	}
	if claims.IssuedAt > now.Add(time.Minute).Unix() {
		return Claims{}, errors.New("issued-at is implausibly in the future")
	}
	return claims, nil
}
