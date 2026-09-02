package server

import (
	"context"
	"math"
	"testing"
)

func TestGoRunnerClassicAlgorithms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		algorithm string
		config    map[string]float64
		timeline  []RequestPoint
		allowed   []bool
	}{
		{
			name:      "fixed window rejects third request",
			algorithm: "fixed-window",
			config:    map[string]float64{"limit": 2, "windowMs": 1000},
			timeline:  points(0, 100, 200),
			allowed:   []bool{true, true, false},
		},
		{
			name:      "sliding log rejects third request",
			algorithm: "sliding-window-log",
			config:    map[string]float64{"limit": 2, "windowMs": 1000},
			timeline:  points(0, 100, 200),
			allowed:   []bool{true, true, false},
		},
		{
			name:      "sliding counter carries weighted previous window",
			algorithm: "sliding-window-counter",
			config:    map[string]float64{"limit": 2, "windowMs": 1000},
			timeline:  points(100, 200, 1100),
			allowed:   []bool{true, true, false},
		},
		{
			name:      "token bucket refills",
			algorithm: "token-bucket",
			config:    map[string]float64{"capacity": 2, "ratePerSecond": 1},
			timeline:  points(0, 0, 0, 1000),
			allowed:   []bool{true, true, false, true},
		},
		{
			name:      "leaky bucket drains",
			algorithm: "leaky-bucket",
			config:    map[string]float64{"capacity": 2, "ratePerSecond": 1},
			timeline:  points(0, 0, 0, 1000),
			allowed:   []bool{true, true, false, true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response, err := (GoRunner{}).Run(context.Background(), RunRequest{
				Algorithm:       tt.algorithm,
				Language:        "go",
				Config:          tt.config,
				RequestTimeline: tt.timeline,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if len(response.Decisions) != len(tt.allowed) {
				t.Fatalf("got %d decisions, want %d", len(response.Decisions), len(tt.allowed))
			}
			for i, want := range tt.allowed {
				if got := response.Decisions[i].Allowed; got != want {
					t.Errorf("decision[%d].Allowed = %v, want %v", i, got, want)
				}
			}
			if len(response.Events) < len(response.Decisions) {
				t.Fatalf("got %d trace events for %d decisions", len(response.Events), len(response.Decisions))
			}
			for _, event := range response.Events {
				if event.StepID == "" || event.Source.Path == "" || event.Source.Line <= 0 {
					t.Fatalf("trace event is not source anchored: %+v", event)
				}
			}
		})
	}
}

func TestGoRunnerRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	_, err := (GoRunner{}).Run(context.Background(), RunRequest{
		Algorithm:       "token-bucket",
		Language:        "go",
		Config:          map[string]float64{"capacity": 0, "ratePerSecond": 1},
		RequestTimeline: points(0),
	})
	if err == nil {
		t.Fatal("Run() error = nil, want invalid configuration error")
	}
}

func TestSlidingCounterReportsEarliestRetryAcrossBoundary(t *testing.T) {
	response, err := (GoRunner{}).Run(context.Background(), RunRequest{
		Algorithm: AlgorithmSlidingWindowCount,
		Language:  LanguageGo,
		Config:    map[string]float64{"limit": 2, "windowMs": 1000},
		RequestTimeline: []RequestPoint{
			{AtMs: 100, Cost: 1, Key: "alice"},
			{AtMs: 200, Cost: 1, Key: "alice"},
			{AtMs: 1100, Cost: 1, Key: "alice"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	decision := response.Decisions[2]
	if decision.Allowed || decision.RetryAfterMs != 400 || decision.ResetAtMs != 1500 {
		t.Fatalf("decision = %+v, want earliest retry at t=1500", decision)
	}
}

func TestGoRunnerNeverEmitsNonFiniteDecisionFields(t *testing.T) {
	response, err := (GoRunner{}).Run(context.Background(), RunRequest{
		Algorithm: AlgorithmTokenBucket,
		Language:  LanguageGo,
		Config: map[string]float64{
			"capacity":      float64(maxSafeIntegerMilliseconds),
			"ratePerSecond": 1000,
		},
		RequestTimeline: []RequestPoint{{AtMs: 0, Cost: float64(maxSafeIntegerMilliseconds), Key: "alice"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range response.Decisions {
		for name, value := range map[string]float64{"remaining": decision.Remaining, "retryAfterMs": decision.RetryAfterMs, "resetAtMs": decision.ResetAtMs} {
			if math.IsInf(value, 0) || math.IsNaN(value) {
				t.Fatalf("decision.%s = %v, want finite", name, value)
			}
		}
	}
}

func points(timestamps ...int64) []RequestPoint {
	result := make([]RequestPoint, 0, len(timestamps))
	for _, timestamp := range timestamps {
		result = append(result, RequestPoint{Key: "alice", AtMs: timestamp, Cost: 1})
	}
	return result
}
