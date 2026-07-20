package distributedsemantics

// EventKind identifies the part of a session history represented by an event.
type EventKind string

const (
	WriteAcknowledged EventKind = "write_acknowledged"
	ReadObserved      EventKind = "read_observed"
)

// SessionEvent records a version acknowledged or observed by one client session.
// The caller supplies versions from the system under test; the checker only
// evaluates the ordering contract declared by the history.
type SessionEvent struct {
	SessionID string
	Kind      EventKind
	Version   uint64
}

// Violation describes the first-class counterexample returned by a checker.
type Violation struct {
	Property        string
	SessionID       string
	EventIndex      int
	ObservedVersion uint64
	RequiredVersion uint64
}

// CheckReadYourWrites reports reads that observe a version older than a write
// already acknowledged in the same session.
func CheckReadYourWrites(history []SessionEvent) []Violation {
	latestWrite := make(map[string]uint64)
	violations := make([]Violation, 0)

	for i, event := range history {
		switch event.Kind {
		case WriteAcknowledged:
			if event.Version > latestWrite[event.SessionID] {
				latestWrite[event.SessionID] = event.Version
			}
		case ReadObserved:
			required := latestWrite[event.SessionID]
			if event.Version < required {
				violations = append(violations, Violation{
					Property:        "read-your-writes",
					SessionID:       event.SessionID,
					EventIndex:      i,
					ObservedVersion: event.Version,
					RequiredVersion: required,
				})
			}
		}
	}

	return violations
}

// CheckMonotonicReads reports reads that move backwards within one session.
func CheckMonotonicReads(history []SessionEvent) []Violation {
	latestRead := make(map[string]uint64)
	seenRead := make(map[string]bool)
	violations := make([]Violation, 0)

	for i, event := range history {
		if event.Kind != ReadObserved {
			continue
		}

		previous := latestRead[event.SessionID]
		if seenRead[event.SessionID] && event.Version < previous {
			violations = append(violations, Violation{
				Property:        "monotonic-reads",
				SessionID:       event.SessionID,
				EventIndex:      i,
				ObservedVersion: event.Version,
				RequiredVersion: previous,
			})
		}

		if !seenRead[event.SessionID] || event.Version > previous {
			latestRead[event.SessionID] = event.Version
		}
		seenRead[event.SessionID] = true
	}

	return violations
}
