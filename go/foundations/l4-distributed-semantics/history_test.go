package distributedsemantics

import "testing"

func TestCheckReadYourWrites(t *testing.T) {
	history := []SessionEvent{
		{SessionID: "operator", Kind: WriteAcknowledged, Version: 43},
		{SessionID: "other", Kind: ReadObserved, Version: 41},
		{SessionID: "operator", Kind: ReadObserved, Version: 42},
		{SessionID: "operator", Kind: ReadObserved, Version: 43},
	}

	violations := CheckReadYourWrites(history)
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want exactly one", violations)
	}
	got := violations[0]
	if got.SessionID != "operator" || got.EventIndex != 2 || got.ObservedVersion != 42 || got.RequiredVersion != 43 {
		t.Fatalf("violation = %+v, want operator event 2: observed 42, required 43", got)
	}
}

func TestCheckReadYourWritesAllowsIndependentSessions(t *testing.T) {
	history := []SessionEvent{
		{SessionID: "a", Kind: WriteAcknowledged, Version: 7},
		{SessionID: "b", Kind: ReadObserved, Version: 3},
		{SessionID: "a", Kind: ReadObserved, Version: 7},
	}

	if violations := CheckReadYourWrites(history); len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
}

func TestCheckMonotonicReads(t *testing.T) {
	history := []SessionEvent{
		{SessionID: "a", Kind: ReadObserved, Version: 10},
		{SessionID: "b", Kind: ReadObserved, Version: 2},
		{SessionID: "a", Kind: ReadObserved, Version: 8},
		{SessionID: "a", Kind: ReadObserved, Version: 11},
	}

	violations := CheckMonotonicReads(history)
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want exactly one", violations)
	}
	got := violations[0]
	if got.EventIndex != 2 || got.ObservedVersion != 8 || got.RequiredVersion != 10 {
		t.Fatalf("violation = %+v, want event 2: observed 8, required 10", got)
	}
}

func TestCheckMonotonicReadsAcceptsEqualVersions(t *testing.T) {
	history := []SessionEvent{
		{SessionID: "a", Kind: ReadObserved, Version: 4},
		{SessionID: "a", Kind: ReadObserved, Version: 4},
		{SessionID: "a", Kind: ReadObserved, Version: 5},
	}

	if violations := CheckMonotonicReads(history); len(violations) != 0 {
		t.Fatalf("violations = %v, want none", violations)
	}
}
