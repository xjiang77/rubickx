package lab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
	"github.com/xjiang77/rubickx/network-security/internal/oidclab"
)

func TestOIDCNegativeMatrixRejectsBeforeRPSession(t *testing.T) {
	const issuer = "http://127.0.0.1/oidc-test"
	const clientID = "rp-a"
	const redirectURI = "http://127.0.0.1/callback"
	provider, err := oidclab.NewProvider(issuer)
	if err != nil {
		t.Fatal(err)
	}
	provider.RegisterClient(clientID, redirectURI)
	cases, err := buildOIDCNegativeMatrix(provider, issuer, clientID, redirectURI, "nonce-1", "verifier-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{
		"unregistered_redirect": true, "missing_state": true, "missing_nonce": true,
		"wrong_pkce": true, "code_replay": true, "code_injection": true,
		"issuer": true, "audience": true, "nonce": true, "expiry": true, "unknown_kid": true,
	}
	rpSessions := 0
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				rpSessions++
				t.Fatalf("%s unexpectedly passed pre-session validation", test.name)
			}
			if rpSessions != 0 {
				t.Fatalf("RP sessions=%d after %s", rpSessions, test.name)
			}
			delete(wanted, test.name)
		})
	}
	for name := range wanted {
		t.Errorf("negative matrix missing %s", name)
	}
}

func TestRouteTableLongestPrefixAndIndependentFailures(t *testing.T) {
	table := routeTable{entries: []routeEntry{
		{prefix: netip.MustParsePrefix("127.0.0.0/8"), nextHop: "loopback", pathMTU: 1280},
		{prefix: netip.MustParsePrefix("127.0.0.1/32"), nextHop: "service", pathMTU: 900},
	}}
	selected, err := table.admit(netip.MustParseAddr("127.0.0.1"), 512)
	if err != nil {
		t.Fatal(err)
	}
	if selected.prefix.Bits() != 32 || selected.nextHop != "service" {
		t.Fatalf("selected route=%s via %s, want /32 service", selected.prefix, selected.nextHop)
	}
	if _, err := table.admit(netip.MustParseAddr("192.0.2.10"), 512); !errors.Is(err, errNoRoute) {
		t.Fatalf("no-route error=%v", err)
	}
	if _, err := table.admit(netip.MustParseAddr("127.0.0.1"), 901); !errors.Is(err, errPathMTUExceeded) {
		t.Fatalf("PMTU error=%v", err)
	}
}

func TestDeadlineReconciliationCommitsBeforeTimeoutWithoutDuplicateEffect(t *testing.T) {
	result, err := executeDeadlineReconciliation("request-42")
	if err != nil {
		t.Fatal(err)
	}
	if !result.ClientTimedOut || !result.ServerCommitted {
		t.Fatalf("timeout=%t committed=%t", result.ClientTimedOut, result.ServerCommitted)
	}
	if result.Applications != 1 || result.ReconciledResult != "result-request-42" {
		t.Fatalf("applications=%d result=%q", result.Applications, result.ReconciledResult)
	}
}

func TestFederationVerifierBindsExpectedIssuer(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	assertion := federationAssertion{
		ID: "issuer-binding", Issuer: "issuer-a", Subject: "subject-1", Audience: "rp-a",
		Recipient: "http://127.0.0.1/acs", InResponseTo: "request-1",
		IssuedAt: now.Add(-time.Second).Unix(), ExpiresAt: now.Add(time.Minute).Unix(),
	}
	token, err := signFederationAssertion(assertion, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	if _, err := verifyFederationAssertion(token, publicKey, "issuer-b", "rp-a", assertion.Recipient, assertion.InResponseTo, now, seen); err == nil {
		t.Fatal("wrong issuer unexpectedly verified")
	}
	if seen[assertion.ID] {
		t.Fatal("failed issuer binding consumed the replay identifier")
	}
}

func TestSignalReceiverRejectsUnknownEventTypeWithoutChangingState(t *testing.T) {
	now := time.Now()
	receiver := newSignalReceiver()
	receiver.active = true
	unknown := securityEventEnvelope{
		JTI: "jti-unknown", EventType: "unknown-event", Subject: "issuer-a|subject-1",
		DeviceID: "device-1", IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(time.Minute), Compliant: false,
	}
	if err := receiver.apply(unknown, now); err == nil {
		t.Fatal("unknown event type unexpectedly applied")
	}
	if !receiver.active || receiver.seen[unknown.JTI] {
		t.Fatalf("unknown event changed state: active=%t seen=%t", receiver.active, receiver.seen[unknown.JTI])
	}
}

func TestCapstoneExecutesTelemetryToReadbackChain(t *testing.T) {
	result, err := Run(context.Background(), "LAB-NETSEC-10")
	if err != nil {
		t.Fatal(err)
	}
	byOutcome := make(map[string]evidence.Event)
	for _, event := range result.Events {
		byOutcome[event.Outcome] = event
	}
	normal := byOutcome[evidence.Normal]
	if normal.Stage != "evidence_closure" || normal.Component != "access-controller" {
		t.Fatalf("normal stage/component=%s/%s", normal.Stage, normal.Component)
	}
	for _, evidenceToken := range []string{"telemetry_compliant=true", "signal=device-noncompliance:false", "decision=allow", "readback=3/3"} {
		if !strings.Contains(normal.ObservedState, evidenceToken) {
			t.Errorf("normal evidence missing %q: %s", evidenceToken, normal.ObservedState)
		}
	}
	recovery := byOutcome[evidence.Recovery]
	if !strings.Contains(recovery.ObservedState, "readback=3/3") {
		t.Errorf("recovery lacks effective readback: %s", recovery.ObservedState)
	}
}

func TestEventCarriesScenarioEvidence(t *testing.T) {
	event := evidence.NewEvent("LAB-NETSEC-01", evidence.Reject, "deny", "route rejected")
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"scenario_id", "stage", "component", "evidence_kind", "observed_state",
		"action_id", "precondition_revision",
	} {
		value, ok := fields[name].(string)
		if !ok || value == "" {
			t.Errorf("event field %s is missing: %s", name, raw)
		}
	}
}

func TestEveryLabExecutesEveryOutcome(t *testing.T) {
	t.Parallel()
	report, err := RunAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Labs) != 10 {
		t.Fatalf("got %d labs, want 10", len(report.Labs))
	}
	if report.SchemaVersion != 2 {
		t.Fatalf("schema_version=%d, want 2", report.SchemaVersion)
	}
	for _, result := range report.Labs {
		if len(result.Events) != len(evidence.RequiredOutcomes) {
			t.Errorf("%s events=%d, want %d", result.ID, len(result.Events), len(evidence.RequiredOutcomes))
		}
		seen := make(map[string]bool)
		for _, event := range result.Events {
			if seen[event.Outcome] {
				t.Errorf("%s repeats outcome %s", result.ID, event.Outcome)
			}
			seen[event.Outcome] = true
			if event.TraceID == "" || event.RequestID == "" || event.Decision == "" || event.Reason == "" {
				t.Fatalf("%s has incomplete event: %#v", result.ID, event)
			}
			if event.DesiredState == "" || event.EffectiveState == "" || event.Ack == "" {
				t.Fatalf("%s loses state/evidence boundary: %#v", result.ID, event)
			}
			if event.ScenarioID == "" || event.Stage == "" || event.Component == "" || event.EvidenceKind == "" || event.ObservedState == "" || event.ActionID == "" || event.PreconditionRevision == "" {
				t.Fatalf("%s has incomplete scenario evidence: %#v", result.ID, event)
			}
		}
		for _, outcome := range evidence.RequiredOutcomes {
			if !seen[outcome] {
				t.Errorf("%s missing %s", result.ID, outcome)
			}
		}
	}
}

func TestCoreLabsExposeExecutedScenarioComponents(t *testing.T) {
	core := map[string]bool{
		"LAB-NETSEC-01": true,
		"LAB-NETSEC-02": true,
		"LAB-NETSEC-03": true,
		"LAB-NETSEC-04": true,
		"LAB-NETSEC-07": true,
		"LAB-NETSEC-08": true,
		"LAB-NETSEC-09": true,
		"LAB-NETSEC-10": true,
	}
	report, err := RunAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range report.Labs {
		if !core[result.ID] {
			continue
		}
		seen := make(map[string]bool, len(result.Events))
		for _, event := range result.Events {
			if event.Component == "lab-runner" {
				t.Errorf("%s %s uses generic component", result.ID, event.Outcome)
			}
			if seen[event.ScenarioID] {
				t.Errorf("%s reuses scenario id %q", result.ID, event.ScenarioID)
			}
			seen[event.ScenarioID] = true
		}
	}
}

func TestUnknownLabFails(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), "LAB-NETSEC-99"); err == nil {
		t.Fatal("unknown lab unexpectedly succeeded")
	}
}
