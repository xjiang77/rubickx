package server

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDelveSessionManagerRunsRealDAPSession(t *testing.T) {
	if _, err := exec.LookPath("dlv"); err != nil {
		t.Skip("dlv is not installed")
	}
	root, err := LabRoot()
	if err != nil {
		t.Fatal(err)
	}
	manager := NewDelveSessionManager(root)
	t.Cleanup(func() { _ = manager.CloseAll() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	snapshot, err := manager.Create(ctx, DebugSessionRequest{
		Algorithm:        AlgorithmTokenBucket,
		Config:           map[string]float64{"capacity": 2, "ratePerSecond": 1},
		RequestTimeline:  points(0, 0, 1000),
		BreakpointStepID: "token.refill",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snapshot.SessionID == "" || snapshot.Status != "stopped" || snapshot.Line <= 0 || len(snapshot.StackFrames) == 0 || len(snapshot.Locals) == 0 {
		t.Fatalf("incomplete initial snapshot: %+v", snapshot)
	}
	if snapshot.Source.Path != "server/algorithms.go" || !strings.Contains(snapshot.Source.Content, "@step:token.refill") {
		t.Fatalf("debugger did not stop in the real algorithm source: %+v", snapshot.Source)
	}
	localNames := map[string]bool{}
	for _, local := range snapshot.Locals {
		localNames[local.Name] = true
	}
	if !localNames["point"] && !localNames["state"] {
		t.Fatalf("algorithm locals are missing from snapshot: %+v", snapshot.Locals)
	}
	wire, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"source":{"path":`) || !strings.Contains(string(wire), `"file":`) {
		t.Fatalf("debug snapshot wire contract is missing source content or frame file: %s", wire)
	}

	next, err := manager.Command(ctx, snapshot.SessionID, "next")
	if err != nil {
		t.Fatalf("next error = %v", err)
	}
	if next.Status != "stopped" || next.Line <= 0 {
		t.Fatalf("next snapshot = %+v", next)
	}

	continued, err := manager.Command(ctx, snapshot.SessionID, "continue")
	if err != nil {
		t.Fatalf("continue error = %v", err)
	}
	if continued.Status != "stopped" || continued.Source.Path != "server/algorithms.go" {
		t.Fatalf("continue snapshot = %+v", continued)
	}

	restarted, err := manager.Command(ctx, snapshot.SessionID, "restart")
	if err != nil {
		t.Fatalf("restart error = %v", err)
	}
	if restarted.Status != "stopped" || restarted.Line <= 0 {
		t.Fatalf("restart snapshot = %+v", restarted)
	}

	if _, err := manager.Command(ctx, snapshot.SessionID, "stop"); err != nil {
		t.Fatalf("stop error = %v", err)
	}
}

func TestDebugStepWhitelistMapsToRealAlgorithmSource(t *testing.T) {
	root, err := LabRoot()
	if err != nil {
		t.Fatal(err)
	}
	source := root + "/server/algorithms.go"
	for algorithm, steps := range debugSteps {
		if len(steps) < 2 {
			t.Errorf("%s exposes no useful debug steps", algorithm)
		}
		for _, step := range steps {
			if line, err := findStepLine(source, step); err != nil || line <= 0 {
				t.Errorf("%s step %s has no real source anchor: line=%d err=%v", algorithm, step, line, err)
			}
		}
	}

	manager := NewDelveSessionManager(root)
	_, err = manager.Create(context.Background(), DebugSessionRequest{
		Algorithm:        AlgorithmTokenBucket,
		Config:           map[string]float64{"capacity": 1, "ratePerSecond": 1},
		RequestTimeline:  points(0),
		BreakpointStepID: "fixed.decision",
	})
	if err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("mismatched breakpoint error = %v", err)
	}
}
