package evidence

import (
	"fmt"
	"time"
)

const (
	Normal            = "normal"
	Reject            = "reject"
	DependencyFailure = "dependency_failure"
	Degraded          = "degraded"
	Recovery          = "recovery"
)

var RequiredOutcomes = []string{Normal, Reject, DependencyFailure, Degraded, Recovery}

type Event struct {
	Timestamp            string `json:"timestamp"`
	LabID                string `json:"lab_id"`
	ScenarioID           string `json:"scenario_id"`
	Stage                string `json:"stage"`
	Component            string `json:"component"`
	EvidenceKind         string `json:"evidence_kind"`
	ObservedState        string `json:"observed_state"`
	ActionID             string `json:"action_id"`
	PreconditionRevision string `json:"precondition_revision"`
	Outcome              string `json:"outcome"`
	TraceID              string `json:"trace_id"`
	RequestID            string `json:"request_id"`
	SessionID            string `json:"session_id"`
	Issuer               string `json:"issuer"`
	Subject              string `json:"subject"`
	DeviceID             string `json:"device_id"`
	PolicyRevision       string `json:"policy_revision"`
	Decision             string `json:"decision"`
	Reason               string `json:"reason"`
	DesiredState         string `json:"desired_state"`
	EffectiveState       string `json:"effective_state"`
	Action               string `json:"action"`
	Ack                  string `json:"ack"`
}

type LabResult struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Outcomes []string `json:"outcomes"`
	Events   []Event  `json:"events"`
}

type Report struct {
	SchemaVersion int         `json:"schema_version"`
	GeneratedAt   string      `json:"generated_at"`
	Labs          []LabResult `json:"labs"`
}

func NewEvent(labID, outcome, decision, reason string) Event {
	suffix := labID[len(labID)-2:]
	return Event{
		Timestamp:            time.Now().UTC().Format(time.RFC3339Nano),
		LabID:                labID,
		ScenarioID:           fmt.Sprintf("%s-%s-scenario", labID, outcome),
		Stage:                "scenario_assertion",
		Component:            "lab-runner",
		EvidenceKind:         "assertion",
		ObservedState:        decision,
		ActionID:             fmt.Sprintf("action-netsec-%s-%s", suffix, outcome),
		PreconditionRevision: "policy-v1",
		Outcome:              outcome,
		TraceID:              fmt.Sprintf("trace-netsec-%s-%s", suffix, outcome),
		RequestID:            fmt.Sprintf("request-netsec-%s-%s", suffix, outcome),
		SessionID:            "n/a",
		Issuer:               "n/a",
		Subject:              "n/a",
		DeviceID:             "device-localhost",
		PolicyRevision:       "policy-v1",
		Decision:             decision,
		Reason:               reason,
		DesiredState:         "allow only when the lab invariant is satisfied",
		EffectiveState:       decision,
		Action:               "record",
		Ack:                  "observed",
	}
}

func NewResult(id, title string, events []Event) (LabResult, error) {
	seen := make(map[string]bool, len(events))
	for _, event := range events {
		if event.ScenarioID == "" || event.Stage == "" || event.Component == "" || event.EvidenceKind == "" ||
			event.ObservedState == "" || event.ActionID == "" || event.PreconditionRevision == "" {
			return LabResult{}, fmt.Errorf("%s has incomplete scenario evidence for %s", id, event.Outcome)
		}
		seen[event.Outcome] = true
	}
	for _, outcome := range RequiredOutcomes {
		if !seen[outcome] {
			return LabResult{}, fmt.Errorf("%s missing outcome %s", id, outcome)
		}
	}
	return LabResult{ID: id, Title: title, Outcomes: append([]string(nil), RequiredOutcomes...), Events: events}, nil
}
