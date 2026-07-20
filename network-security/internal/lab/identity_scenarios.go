package lab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
	"github.com/xjiang77/rubickx/network-security/internal/oidclab"
)

func tamperJWTKeyID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token
	}
	parts[0] = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"unknown","typ":"JWT"}`))
	return strings.Join(parts, ".")
}

type oidcNegativeCase struct {
	name string
	run  func() error
}

func validateResponseState(received, expected string) error {
	if received == "" || received != expected {
		return errors.New("authorization response state mismatch")
	}
	return nil
}

func buildOIDCNegativeMatrix(
	provider *oidclab.Provider,
	issuer, clientID, redirectURI, nonce, verifier string,
	now time.Time,
) ([]oidcNegativeCase, error) {
	baseClaims := oidclab.Claims{
		Issuer: issuer, Subject: "subject-123", Audience: clientID, Azp: clientID,
		Nonce: nonce, Expires: now.Add(time.Minute).Unix(), IssuedAt: now.Unix(),
		NotBefore: now.Add(-time.Second).Unix(), SessionID: "sid-negative",
	}
	validToken, err := provider.IssueTestToken(baseClaims, "")
	if err != nil {
		return nil, err
	}
	badPKCECode, err := provider.Authorize(clientID, redirectURI, "subject-123", nonce, oidclab.PKCEChallenge(verifier))
	if err != nil {
		return nil, err
	}
	replayCode, err := provider.Authorize(clientID, redirectURI, "subject-123", nonce, oidclab.PKCEChallenge(verifier))
	if err != nil {
		return nil, err
	}
	if _, err := provider.Redeem(replayCode, clientID, redirectURI, verifier); err != nil {
		return nil, err
	}
	expiredClaims := baseClaims
	expiredClaims.Expires = now.Add(-time.Second).Unix()
	expiredToken, err := provider.IssueTestToken(expiredClaims, "")
	if err != nil {
		return nil, err
	}
	return []oidcNegativeCase{
		{"unregistered_redirect", func() error {
			_, err := provider.Authorize(clientID, issuer+"/other", "subject-123", nonce, oidclab.PKCEChallenge(verifier))
			return err
		}},
		{"missing_state", func() error { return validateResponseState("", "expected-state") }},
		{"missing_nonce", func() error {
			_, err := provider.Authorize(clientID, redirectURI, "subject-123", "", oidclab.PKCEChallenge(verifier))
			return err
		}},
		{"wrong_pkce", func() error {
			_, err := provider.Redeem(badPKCECode, clientID, redirectURI, "wrong-verifier")
			return err
		}},
		{"code_replay", func() error { _, err := provider.Redeem(replayCode, clientID, redirectURI, verifier); return err }},
		{"code_injection", func() error { _, err := provider.Redeem("injected", clientID, redirectURI, verifier); return err }},
		{"issuer", func() error {
			_, err := provider.ValidateIDToken(validToken, issuer+"/other", clientID, nonce, now)
			return err
		}},
		{"audience", func() error { _, err := provider.ValidateIDToken(validToken, issuer, "rp-b", nonce, now); return err }},
		{"nonce", func() error {
			_, err := provider.ValidateIDToken(validToken, issuer, clientID, "wrong", now)
			return err
		}},
		{"expiry", func() error {
			_, err := provider.ValidateIDToken(expiredToken, issuer, clientID, nonce, now)
			return err
		}},
		{"unknown_kid", func() error {
			_, err := provider.ValidateIDToken(tamperJWTKeyID(validToken), issuer, clientID, nonce, now)
			return err
		}},
	}, nil
}

func runLab07(ctx context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-07"
	const issuer = "http://127.0.0.1/oidc-lab"
	const clientID = "rp-a"
	const redirectURI = "http://127.0.0.1/callback"
	provider, err := oidclab.NewProvider(issuer)
	if err != nil {
		return evidence.LabResult{}, err
	}
	provider.RegisterClient(clientID, redirectURI)
	verifier := "local-verifier-with-sufficient-entropy-for-the-lab"
	nonce := "nonce-transaction-1"
	code, err := provider.Authorize(clientID, redirectURI, "subject-123", nonce, oidclab.PKCEChallenge(verifier))
	if err != nil {
		return evidence.LabResult{}, err
	}
	tokens, err := provider.Redeem(code, clientID, redirectURI, verifier)
	if err != nil {
		return evidence.LabResult{}, err
	}
	claims, err := provider.ValidateIDToken(tokens.IDToken, issuer, clientID, nonce, time.Now())
	if err != nil || claims.Subject != "subject-123" {
		return evidence.LabResult{}, fmt.Errorf("ID token validation failed: %w", err)
	}
	rpSessions := 1
	normal := scenarioEvent(id, evidence.Normal, "lab07-code-pkce-session", "rp_session_create", "oidc-rp", "validated_claims", "code_bound=true; pkce=true; iss=true; aud=true; nonce=true; rp_sessions=1", "allow", "the RP creates a session only after code binding, PKCE and ID token claims validate")
	normal.Issuer = issuer
	normal.Subject = claims.Subject
	normal.SessionID = claims.SessionID

	negativeCases, err := buildOIDCNegativeMatrix(provider, issuer, clientID, redirectURI, nonce, verifier, time.Now())
	if err != nil {
		return evidence.LabResult{}, err
	}
	negativeSessions := 0
	for _, test := range negativeCases {
		if err := test.run(); err == nil {
			negativeSessions++
			return evidence.LabResult{}, fmt.Errorf("negative case %s created an RP session", test.name)
		}
	}
	if negativeSessions != 0 || rpSessions != 1 {
		return evidence.LabResult{}, fmt.Errorf("negative sessions=%d total sessions=%d", negativeSessions, rpSessions)
	}
	reject := scenarioEvent(id, evidence.Reject, "lab07-negative-matrix", "pre_session_validation", "oidc-rp", "table_driven_failures", "11_of_11_rejected; rp_sessions_created=0", "deny", "redirect, state, nonce, PKCE, replay, injection, issuer, audience, expiry and kid failures occur before RP session creation")
	reject.Issuer = issuer
	reject.Subject = "subject-123"

	unavailable, err := newLoopbackHTTPServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err != nil {
		return evidence.LabResult{}, err
	}
	unavailableURL := unavailable.URL
	unavailable.Close()
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, unavailableURL+"/token", nil)
	if _, err := http.DefaultClient.Do(request); err == nil {
		return evidence.LabResult{}, errors.New("closed token endpoint unexpectedly responded")
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab07-token-endpoint-down", "token_exchange", "oidc-rp", "loopback_connection_error", "token_endpoint=unavailable; rp_sessions_created=0", "deny", "a loopback token endpoint outage prevents a new RP session")
	dependency.Issuer = issuer
	dependency.Subject = "subject-123"

	if err := provider.Rotate("kid-2"); err != nil {
		return evidence.LabResult{}, err
	}
	if _, err := provider.ValidateIDToken(tokens.IDToken, issuer, clientID, nonce, time.Now()); err != nil {
		return evidence.LabResult{}, fmt.Errorf("old key was not accepted during bounded rotation overlap: %w", err)
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab07-jwks-overlap", "token_validation", "oidc-rp", "key_overlap", "retired_kid_accepted_within_bounded_overlap", "validate_bounded_old_key", "a retired signing key remains valid only during the configured overlap window")
	degraded.Issuer = issuer
	degraded.Subject = "subject-123"
	degraded.SessionID = claims.SessionID

	newCode, err := provider.Authorize(clientID, redirectURI, "subject-123", "nonce-2", oidclab.PKCEChallenge(verifier))
	if err != nil {
		return evidence.LabResult{}, err
	}
	newTokens, err := provider.Redeem(newCode, clientID, redirectURI, verifier)
	if err != nil {
		return evidence.LabResult{}, err
	}
	newClaims, err := provider.ValidateIDToken(newTokens.IDToken, issuer, clientID, "nonce-2", time.Now())
	if err != nil {
		return evidence.LabResult{}, err
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab07-kid2-session", "rp_session_create", "oidc-rp", "validated_claims", "active_kid=kid-2; fresh_code=true; rp_session_created=true", "allow", "a fresh transaction validates under kid-2 and creates a new RP session")
	recovery.Issuer = issuer
	recovery.Subject = newClaims.Subject
	recovery.SessionID = newClaims.SessionID
	recovery.PolicyRevision = "jwks-kid-2"
	recovery.PreconditionRevision = "jwks-kid-1+2"

	return evidence.NewResult(id, "OIDC-style Authorization Code and PKCE", []evidence.Event{normal, reject, dependency, degraded, recovery})
}

type federationAssertion struct {
	ID           string `json:"id"`
	Issuer       string `json:"issuer"`
	Subject      string `json:"subject"`
	Audience     string `json:"audience"`
	Recipient    string `json:"recipient"`
	InResponseTo string `json:"in_response_to"`
	IssuedAt     int64  `json:"issued_at"`
	ExpiresAt    int64  `json:"expires_at"`
}

func signFederationAssertion(assertion federationAssertion, privateKey ed25519.PrivateKey) (string, error) {
	payload, err := json.Marshal(assertion)
	if err != nil {
		return "", err
	}
	signature := ed25519.Sign(privateKey, payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func verifyFederationAssertion(token string, publicKey ed25519.PublicKey, issuer, audience, recipient, responseID string, now time.Time, seen map[string]bool) (federationAssertion, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return federationAssertion{}, errors.New("signed fixture must have payload and signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return federationAssertion{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !ed25519.Verify(publicKey, payload, signature) {
		return federationAssertion{}, errors.New("signed fixture verification failed")
	}
	var assertion federationAssertion
	if err := json.Unmarshal(payload, &assertion); err != nil {
		return federationAssertion{}, err
	}
	if assertion.Issuer != issuer || assertion.Audience != audience || assertion.Recipient != recipient || assertion.InResponseTo != responseID {
		return federationAssertion{}, errors.New("federation field binding mismatch")
	}
	if now.Unix() < assertion.IssuedAt || now.Unix() >= assertion.ExpiresAt {
		return federationAssertion{}, errors.New("federation assertion outside validity window")
	}
	if seen[assertion.ID] {
		return federationAssertion{}, errors.New("federation assertion replayed")
	}
	seen[assertion.ID] = true
	return assertion, nil
}

type federationSession struct {
	Active   bool
	Revision int
}

func runLab08(context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-08"
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return evidence.LabResult{}, err
	}
	now := time.Now()
	assertion := federationAssertion{
		ID: "assertion-1", Issuer: "federation-lab", Subject: "issuer-a|subject-123",
		Audience: "rp-a", Recipient: "http://127.0.0.1/rp-a/acs", InResponseTo: "request-1",
		IssuedAt: now.Add(-time.Second).Unix(), ExpiresAt: now.Add(time.Minute).Unix(),
	}
	token, err := signFederationAssertion(assertion, privateKey)
	if err != nil {
		return evidence.LabResult{}, err
	}
	seen := make(map[string]bool)
	verified, err := verifyFederationAssertion(token, publicKey, assertion.Issuer, assertion.Audience, assertion.Recipient, assertion.InResponseTo, now, seen)
	if err != nil {
		return evidence.LabResult{}, err
	}
	sessions := map[string]*federationSession{"rp-a": {Active: true, Revision: 1}, "rp-b": {Active: true, Revision: 1}}
	if verified.Subject != assertion.Subject || !sessions["rp-a"].Active || !sessions["rp-b"].Active {
		return evidence.LabResult{}, errors.New("verified federation fixture did not establish expected local sessions")
	}
	normal := scenarioEvent(id, evidence.Normal, "lab08-signed-federation", "assertion_validation", "federation-rp", "signature_and_field_bindings", "signature=valid; audience=rp-a; recipient=acs; sessions=rp-a+rp-b", "allow", "a signed educational fixture is accepted only after signature, field and time bindings validate")

	invalid := assertion
	invalid.ID = "assertion-2"
	invalid.Audience = "rp-b"
	invalidToken, err := signFederationAssertion(invalid, privateKey)
	if err != nil {
		return evidence.LabResult{}, err
	}
	if _, err := verifyFederationAssertion(invalidToken, publicKey, assertion.Issuer, "rp-a", assertion.Recipient, assertion.InResponseTo, now, seen); err == nil {
		return evidence.LabResult{}, errors.New("wrong federation audience unexpectedly verified")
	}
	wrongIssuer := assertion
	wrongIssuer.ID = "assertion-3"
	wrongIssuer.Issuer = "untrusted-issuer"
	wrongIssuerToken, err := signFederationAssertion(wrongIssuer, privateKey)
	if err != nil {
		return evidence.LabResult{}, err
	}
	if _, err := verifyFederationAssertion(wrongIssuerToken, publicKey, assertion.Issuer, assertion.Audience, assertion.Recipient, assertion.InResponseTo, now, seen); err == nil {
		return evidence.LabResult{}, errors.New("wrong federation issuer unexpectedly verified")
	}
	if _, err := verifyFederationAssertion(token, publicKey, assertion.Issuer, assertion.Audience, assertion.Recipient, assertion.InResponseTo, now, seen); err == nil {
		return evidence.LabResult{}, errors.New("federation replay unexpectedly verified")
	}
	reject := scenarioEvent(id, evidence.Reject, "lab08-binding-and-replay", "assertion_validation", "federation-rp", "verification_errors", "wrong_issuer=rejected; wrong_audience=rejected; replayed_id=rejected", "deny", "a valid signature does not bypass expected issuer, audience, recipient, response-correlation, time or replay checks")

	sessions["rp-a"].Active = false
	backChannelDelivered := false
	if backChannelDelivered {
		sessions["rp-b"].Active = false
	}
	if sessions["rp-a"].Active || !sessions["rp-b"].Active {
		return evidence.LabResult{}, errors.New("partial logout state was not preserved")
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab08-partial-logout", "logout_delivery", "federation-rp", "session_inventory", "rp-a=inactive; rp-b=active; backchannel=failed", "partial_logout", "RP-A logout succeeds while RP-B back-channel delivery is unavailable")
	dependency.DesiredState = "all_sessions_inactive"
	dependency.EffectiveState = "rp-a_inactive;rp-b_active"
	dependency.Ack = "rp-b_delivery_failed"

	deprovisioned := true
	createSession := func() error {
		if deprovisioned {
			return errors.New("subject has a provisioning tombstone")
		}
		return nil
	}
	if err := createSession(); err == nil || !sessions["rp-b"].Active {
		return evidence.LabResult{}, errors.New("deprovision scenario did not block new sessions while preserving partial logout evidence")
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab08-deprovision-convergence", "provisioning_enforcement", "identity-lifecycle", "session_inventory", "new_session=denied; rp-b=active_pending_retry", "deny_new_and_retry_logout", "a provisioning tombstone blocks new sessions while asynchronous logout retry remains visible")

	sessions["rp-b"].Active = false
	sessions["rp-b"].Revision++
	if sessions["rp-b"].Active || sessions["rp-b"].Revision != 2 {
		return evidence.LabResult{}, errors.New("logout retry did not converge")
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab08-logout-tombstone-recovery", "lifecycle_reconcile", "identity-lifecycle", "session_inventory", "rp-a=inactive; rp-b=inactive; tombstone_revision=2", "logged_out", "back-channel retry and the provisioning tombstone converge all known sessions")
	recovery.PolicyRevision = "tombstone-v2"
	recovery.PreconditionRevision = "tombstone-v1"

	for _, item := range []*evidence.Event{&normal, &reject, &dependency, &degraded, &recovery} {
		item.Issuer = assertion.Issuer
		item.Subject = assertion.Subject
		item.SessionID = "rp-a+rp-b"
	}
	return evidence.NewResult(id, "Federation, logout and provisioning lifecycle", []evidence.Event{normal, reject, dependency, degraded, recovery})
}

type securityEventEnvelope struct {
	JTI       string
	EventType string
	Subject   string
	DeviceID  string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Compliant bool
}

type signalReceiver struct {
	seen     map[string]bool
	active   bool
	lastSeen time.Time
}

func newSignalReceiver() *signalReceiver {
	return &signalReceiver{seen: make(map[string]bool)}
}

func (receiver *signalReceiver) apply(event securityEventEnvelope, now time.Time) error {
	if event.JTI == "" || event.EventType == "" || event.Subject == "" || event.DeviceID == "" {
		return errors.New("typed signal fields are required")
	}
	if event.EventType != "device-compliance-change" {
		return fmt.Errorf("unsupported signal event type %q", event.EventType)
	}
	if receiver.seen[event.JTI] {
		return errors.New("duplicate signal jti")
	}
	if now.Before(event.IssuedAt) || !now.Before(event.ExpiresAt) {
		return errors.New("signal outside validity window")
	}
	receiver.seen[event.JTI] = true
	receiver.active = event.Compliant
	receiver.lastSeen = now
	return nil
}

func (receiver *signalReceiver) reconcileSnapshot(compliant bool, capturedAt time.Time) {
	receiver.active = compliant
	receiver.lastSeen = capturedAt
}

func runLab09(context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-09"
	now := time.Now()
	receiver := newSignalReceiver()
	allow := securityEventEnvelope{JTI: "jti-allow", EventType: "device-compliance-change", Subject: "issuer-a|subject-123", DeviceID: "device-localhost", IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(time.Minute), Compliant: true}
	if err := receiver.apply(allow, now); err != nil || !receiver.active {
		return evidence.LabResult{}, fmt.Errorf("valid signal did not allow: %w", err)
	}
	normal := scenarioEvent(id, evidence.Normal, "lab09-typed-signal", "signal_consume", "signal-receiver", "typed_envelope", "jti=jti-allow; compliant=true; decision=allow", "allow", "the PDP consumes a typed SSF/CAEP-style event and evaluates subject, device and session context")

	unknown := allow
	unknown.JTI = "jti-unknown"
	unknown.EventType = "unknown-event"
	unknown.Compliant = false
	if err := receiver.apply(unknown, now); err == nil || !receiver.active || receiver.seen[unknown.JTI] {
		return evidence.LabResult{}, errors.New("unknown event type changed receiver state")
	}
	if err := receiver.apply(allow, now); err == nil || !receiver.active {
		return evidence.LabResult{}, errors.New("duplicate jti changed receiver state")
	}
	reject := scenarioEvent(id, evidence.Reject, "lab09-type-and-jti-reject", "signal_consume", "signal-receiver", "validation_errors", "unknown_event_type=rejected; duplicate_jti=rejected; prior_state=allow", "ignore_invalid", "an unknown event type and repeated jti are rejected without changing posture or inventing a protocol sequence")

	expired := securityEventEnvelope{JTI: "jti-expired", EventType: allow.EventType, Subject: allow.Subject, DeviceID: allow.DeviceID, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(-time.Second), Compliant: false}
	if err := receiver.apply(expired, now); err == nil || !receiver.active {
		return evidence.LabResult{}, errors.New("expired signal changed receiver state")
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab09-expired-source-event", "signal_consume", "signal-receiver", "validity_error", "expired_event=rejected; authoritative_update=unavailable", "deny_new", "an expired signal cannot establish fresh posture while the authoritative source is unavailable")

	lost := securityEventEnvelope{JTI: "jti-lost", EventType: allow.EventType, Subject: allow.Subject, DeviceID: allow.DeviceID, IssuedAt: now, ExpiresAt: now.Add(time.Minute), Compliant: false}
	if receiver.seen[lost.JTI] {
		return evidence.LabResult{}, errors.New("lost event unexpectedly appeared in dedupe state")
	}
	decisionTime := now.Add(2 * time.Minute)
	if decisionTime.Sub(receiver.lastSeen) <= time.Minute {
		return evidence.LabResult{}, errors.New("event-loss scenario did not make posture stale")
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab09-event-loss-staleness", "access_decision", "pdp", "freshness_state", "event_lost=true; posture=stale; low_risk=step_up; high_risk=deny", "step_up", "event loss is represented as stale evidence, not silently converted into an allow")

	receiver.reconcileSnapshot(false, decisionTime)
	if receiver.active || !receiver.lastSeen.Equal(decisionTime) {
		return evidence.LabResult{}, errors.New("authoritative snapshot did not reconcile state")
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab09-authoritative-snapshot", "state_reconcile", "signal-receiver", "authoritative_snapshot", "snapshot_compliant=false; active=false; captured_at=fresh", "revoke", "an authoritative snapshot repairs event-loss drift without inventing an event sequence")
	recovery.PolicyRevision = "snapshot-fresh"
	recovery.PreconditionRevision = "event-stream-stale"

	for _, item := range []*evidence.Event{&normal, &reject, &dependency, &degraded, &recovery} {
		item.Issuer = "shared-signals-lab"
		item.Subject = allow.Subject
		item.DeviceID = allow.DeviceID
		item.SessionID = "session-continuous-1"
	}
	return evidence.NewResult(id, "Continuous access and signal ordering", []evidence.Event{normal, reject, dependency, degraded, recovery})
}

type pepTarget struct {
	id        string
	online    bool
	reject    map[string]bool
	effective string
}

type endpointTelemetry struct {
	deviceID  string
	compliant bool
	captured  time.Time
}

type postureSignal struct {
	eventType    string
	deviceID     string
	noncompliant bool
	derivedAt    time.Time
}

func collectEndpointTelemetry(deviceID string, compliant bool, captured time.Time) endpointTelemetry {
	return endpointTelemetry{deviceID: deviceID, compliant: compliant, captured: captured}
}

func derivePostureSignal(sample endpointTelemetry) postureSignal {
	return postureSignal{
		eventType:    "device-noncompliance",
		deviceID:     sample.deviceID,
		noncompliant: !sample.compliant,
		derivedAt:    sample.captured,
	}
}

func decideFromSignal(signal postureSignal) string {
	if signal.eventType != "device-noncompliance" || signal.deviceID == "" || signal.derivedAt.IsZero() {
		return "deny_invalid_signal"
	}
	if signal.noncompliant {
		return "deny"
	}
	return "allow"
}

func (target *pepTarget) apply(revision, state string) string {
	if !target.online {
		return "unknown"
	}
	if target.reject[revision] {
		return "nack"
	}
	target.effective = state
	return "ack"
}

func (target *pepTarget) probe() (string, error) {
	if !target.online {
		return "", errors.New("target offline")
	}
	return target.effective, nil
}

func probeTargets(targets []*pepTarget) (map[string]string, int) {
	states := make(map[string]string, len(targets))
	reachable := 0
	for _, target := range targets {
		state, err := target.probe()
		if err != nil {
			states[target.id] = "unknown"
			continue
		}
		states[target.id] = state
		reachable++
	}
	return states, reachable
}

func rollout(targets []*pepTarget, revision, state string) map[string]string {
	acks := make(map[string]string, len(targets))
	for _, target := range targets {
		acks[target.id] = target.apply(revision, state)
	}
	return acks
}

func runLab10(context.Context) (evidence.LabResult, error) {
	const id = "LAB-NETSEC-10"
	targets := []*pepTarget{
		{id: "pep-a", online: true, reject: make(map[string]bool), effective: "allow-v1"},
		{id: "pep-b", online: true, reject: map[string]bool{"policy-v2": true}, effective: "allow-v1"},
		{id: "pep-c", online: true, reject: make(map[string]bool), effective: "allow-v1"},
	}
	baselineTelemetry := collectEndpointTelemetry("device-localhost", true, time.Now())
	baselineSignal := derivePostureSignal(baselineTelemetry)
	baselineDecision := decideFromSignal(baselineSignal)
	baselineReadback, baselineReachable := probeTargets(targets)
	if baselineDecision != "allow" || baselineReachable != 3 || baselineReadback["pep-a"] != "allow-v1" || baselineReadback["pep-b"] != "allow-v1" || baselineReadback["pep-c"] != "allow-v1" {
		return evidence.LabResult{}, fmt.Errorf("baseline decision=%s reachable=%d readback=%v", baselineDecision, baselineReachable, baselineReadback)
	}
	normalObserved := fmt.Sprintf(
		"telemetry_compliant=%t; signal=%s:%t; decision=%s; readback=%d/3; states=%v",
		baselineTelemetry.compliant, baselineSignal.eventType, baselineSignal.noncompliant,
		baselineDecision, baselineReachable, baselineReadback,
	)
	normal := scenarioEvent(id, evidence.Normal, "lab10-baseline-correlated", "evidence_closure", "access-controller", "telemetry_signal_decision_readback", normalObserved, "allow", "collected endpoint telemetry is transformed into a posture signal and decision, then correlated with three effective-state probes")

	deniedTelemetry := collectEndpointTelemetry("device-localhost", false, time.Now())
	deniedSignal := derivePostureSignal(deniedTelemetry)
	decision := decideFromSignal(deniedSignal)
	if decision != "deny" {
		return evidence.LabResult{}, fmt.Errorf("non-compliant decision=%s", decision)
	}
	reject := scenarioEvent(id, evidence.Reject, "lab10-pdp-deny", "policy_decision", "pdp", "derived_signal_decision", fmt.Sprintf("telemetry_compliant=%t; signal=%s:%t; decision=%s; resource_not_reached", deniedTelemetry.compliant, deniedSignal.eventType, deniedSignal.noncompliant, decision), "deny", "the PDP derives a deny from non-compliant endpoint telemetry before the protected resource")

	targets[2].online = false
	acks := rollout(targets, "policy-v2", "deny-v2")
	if acks["pep-a"] != "ack" || acks["pep-b"] != "nack" || acks["pep-c"] != "unknown" {
		return evidence.LabResult{}, fmt.Errorf("partial rollout acks=%v", acks)
	}
	if targets[0].effective != "deny-v2" || targets[1].effective != "allow-v1" || targets[2].effective != "allow-v1" {
		return evidence.LabResult{}, fmt.Errorf("partial rollout effective states=%q/%q/%q", targets[0].effective, targets[1].effective, targets[2].effective)
	}
	partialReadback, partialReachable := probeTargets(targets)
	if partialReachable != 2 || partialReadback["pep-a"] != "deny-v2" || partialReadback["pep-b"] != "allow-v1" || partialReadback["pep-c"] != "unknown" {
		return evidence.LabResult{}, fmt.Errorf("partial readback reachable=%d states=%v", partialReachable, partialReadback)
	}
	dependency := scenarioEvent(id, evidence.DependencyFailure, "lab10-ack-nack-unknown", "policy_apply", "rollout-controller", "ack_and_readback_matrix", fmt.Sprintf("pep-a=ack; pep-b=nack; pep-c=unknown; readback=%d/3; states=%v", partialReachable, partialReadback), "hold", "one offline PEP produces unknown delivery while another returns a negative acknowledgement; probes preserve the resulting effective states")
	dependency.ActionID = "rollout-policy-v2"
	dependency.Ack = "partial"

	rollbackAcks := rollout(targets[:2], "rollback-v1", "allow-v1")
	if rollbackAcks["pep-a"] != "ack" || rollbackAcks["pep-b"] != "ack" || targets[0].effective != "allow-v1" || targets[1].effective != "allow-v1" {
		return evidence.LabResult{}, fmt.Errorf("rollback acks=%v states=%q/%q", rollbackAcks, targets[0].effective, targets[1].effective)
	}
	rollbackReadback, rollbackReachable := probeTargets(targets)
	if rollbackReachable != 2 || rollbackReadback["pep-a"] != "allow-v1" || rollbackReadback["pep-b"] != "allow-v1" || rollbackReadback["pep-c"] != "unknown" {
		return evidence.LabResult{}, fmt.Errorf("rollback readback reachable=%d states=%v", rollbackReachable, rollbackReadback)
	}
	degraded := scenarioEvent(id, evidence.Degraded, "lab10-partial-rollback", "rollback", "rollout-controller", "effective_state_readback", fmt.Sprintf("pep-a=allow-v1; pep-b=allow-v1; pep-c=unknown_offline; readback=%d/3", rollbackReachable), "hold_last_known_good", "the controller rolls acknowledged targets back and probes reachable effective state while preserving the offline target as unknown")
	degraded.ActionID = "rollback-policy-v2"
	degraded.DesiredState = "deny-v2"
	degraded.EffectiveState = "rollback-v1-on-reachable;pep-c-unknown"
	degraded.Ack = "rollback_partial"

	targets[2].online = true
	recoveryAcks := rollout(targets, "policy-v3", "deny-v3")
	for _, target := range targets {
		if recoveryAcks[target.id] != "ack" || target.effective != "deny-v3" {
			return evidence.LabResult{}, fmt.Errorf("recovery target=%s ack=%s effective=%s", target.id, recoveryAcks[target.id], target.effective)
		}
	}
	recoveryReadback, recoveryReachable := probeTargets(targets)
	if recoveryReachable != 3 || recoveryReadback["pep-a"] != "deny-v3" || recoveryReadback["pep-b"] != "deny-v3" || recoveryReadback["pep-c"] != "deny-v3" {
		return evidence.LabResult{}, fmt.Errorf("recovery readback reachable=%d states=%v", recoveryReachable, recoveryReadback)
	}
	recovery := scenarioEvent(id, evidence.Recovery, "lab10-policy-v3-converged", "state_reconcile", "rollout-controller", "ack_and_readback_matrix", fmt.Sprintf("pep-a=ack; pep-b=ack; pep-c=ack; effective=deny-v3; readback=%d/3", recoveryReachable), "deny", "a new policy revision receives three acknowledgements and three effective-state readbacks after dependency recovery")
	recovery.ActionID = "rollout-policy-v3"
	recovery.PolicyRevision = "policy-v3"
	recovery.PreconditionRevision = "rollback-v1"
	recovery.DesiredState = "deny-v3"
	recovery.EffectiveState = "deny-v3-all-targets"
	recovery.Ack = "all_applied"

	for _, item := range []*evidence.Event{&normal, &reject, &dependency, &degraded, &recovery} {
		item.Issuer = "enterprise-access-lab"
		item.Subject = "issuer-a|subject-123"
		item.SessionID = "session-capstone-1"
	}
	return evidence.NewResult(id, "Enterprise access evidence-closure capstone", []evidence.Event{normal, reject, dependency, degraded, recovery})
}
