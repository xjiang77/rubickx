package server

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLanguageRunnersHaveDecisionAndTraceParity(t *testing.T) {
	for _, tool := range []string{"python3", "node", "javac", "java"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required for four-language parity", tool)
		}
	}
	root, err := LabRoot()
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRunnerRegistry(root)
	fixtures := loadCoreParityFixtures(t, root)
	languages := []string{LanguageGo, LanguagePython, LanguageJava, LanguageJavaScript}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			var canonical RunResponse
			for index, language := range languages {
				runner, err := registry.Runner(language)
				if err != nil {
					t.Fatal(err)
				}
				request := fixture.RunRequest
				request.Language = language
				response, err := runner.Run(context.Background(), request)
				if err != nil {
					t.Fatalf("%s Run() error = %v", language, err)
				}
				for _, event := range response.Events {
					if event.Source.Path == "" || event.Source.Line <= 0 {
						t.Fatalf("%s event %q is not source anchored: %+v", language, event.StepID, event.Source)
					}
				}
				assertContinuousDecisionTrace(t, language, response.Events)
				if index == 0 {
					canonical = response
					assertFixtureExpected(t, fixture, response.Decisions)
					continue
				}
				assertDecisionParity(t, language, canonical.Decisions, response.Decisions)
				if len(canonical.Events) != len(response.Events) {
					t.Fatalf("%s emitted %d events, Go emitted %d", language, len(response.Events), len(canonical.Events))
				}
				for eventIndex := range canonical.Events {
					assertTraceEventParity(t, language, eventIndex, canonical.Events[eventIndex], response.Events[eventIndex])
				}
			}
		})
	}
}

func TestLanguageRunnersMatchCanonicalScenarioAdmissions(t *testing.T) {
	for _, tool := range []string{"python3", "node", "javac", "java"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required for four-language scenario verification", tool)
		}
	}
	root, err := LabRoot()
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRunnerRegistry(root)
	for _, scenario := range DefaultCatalog().Scenarios {
		if scenario.Tier != "core" {
			continue
		}
		scenario := scenario
		t.Run(scenario.ID, func(t *testing.T) {
			if scenario.Brief == nil {
				t.Fatal("scenario brief is nil")
			}
			for _, language := range []string{LanguageGo, LanguagePython, LanguageJava, LanguageJavaScript} {
				runner, runnerErr := registry.Runner(language)
				if runnerErr != nil {
					t.Fatal(runnerErr)
				}
				response, runErr := runner.Run(context.Background(), RunRequest{
					ScenarioID:      scenario.ID,
					Algorithm:       scenario.Algorithm,
					Language:        language,
					Config:          scenario.DefaultConfig,
					RequestTimeline: scenario.RequestTimeline,
				})
				if runErr != nil {
					t.Fatalf("%s Run() error = %v", language, runErr)
				}
				got := make([]string, len(response.Decisions))
				for index, decision := range response.Decisions {
					if decision.Allowed {
						got[index] = "allow"
					} else {
						got[index] = "deny"
					}
				}
				if !reflect.DeepEqual(got, scenario.Brief.Expected.Admissions) {
					t.Errorf("%s admissions = %v, catalog = %v", language, got, scenario.Brief.Expected.Admissions)
				}
			}
		})
	}
}

type coreParityFixture struct {
	Name string `json:"name"`
	RunRequest
	ExpectedAllowed          []bool    `json:"expectedAllowed"`
	ExpectedRemaining        []float64 `json:"expectedRemaining"`
	ExpectedLastReason       string    `json:"expectedLastReason"`
	ExpectedLastRetryAfterMs float64   `json:"expectedLastRetryAfterMs"`
	ExpectedLastResetAtMs    *float64  `json:"expectedLastResetAtMs"`
}

func loadCoreParityFixtures(t *testing.T, root string) []coreParityFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "fixtures", "core-parity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []coreParityFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	return fixtures
}

func assertFixtureExpected(t *testing.T, fixture coreParityFixture, decisions []Decision) {
	t.Helper()
	if len(decisions) != len(fixture.ExpectedAllowed) || len(decisions) != len(fixture.ExpectedRemaining) {
		t.Fatalf("fixture expected %d decisions, got %d", len(fixture.ExpectedAllowed), len(decisions))
	}
	for index := range decisions {
		if decisions[index].Allowed != fixture.ExpectedAllowed[index] || math.Abs(decisions[index].Remaining-fixture.ExpectedRemaining[index]) > 1e-6 {
			t.Errorf("decision[%d] = %+v, expected allowed=%v remaining=%v", index, decisions[index], fixture.ExpectedAllowed[index], fixture.ExpectedRemaining[index])
		}
	}
	last := decisions[len(decisions)-1]
	if last.Reason != fixture.ExpectedLastReason || math.Abs(last.RetryAfterMs-fixture.ExpectedLastRetryAfterMs) > 1e-6 {
		t.Errorf("last decision = %+v, expected reason=%q retryAfterMs=%v", last, fixture.ExpectedLastReason, fixture.ExpectedLastRetryAfterMs)
	}
	if fixture.ExpectedLastResetAtMs != nil && math.Abs(last.ResetAtMs-*fixture.ExpectedLastResetAtMs) > 1e-6 {
		t.Errorf("last resetAtMs = %v, expected %v", last.ResetAtMs, *fixture.ExpectedLastResetAtMs)
	}
}

func TestLanguageRunnersShareInputValidation(t *testing.T) {
	root, err := LabRoot()
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRunnerRegistry(root)
	invalid := []RunRequest{
		{Algorithm: AlgorithmFixedWindow, Config: map[string]float64{"limit": 1, "windowMs": 1.5}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmFixedWindow, Config: map[string]float64{"limit": 1, "windowMs": 9_007_199_254_740_992}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 0, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: ""}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: maxSafeIntegerMilliseconds + 1, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmFixedWindow, Config: map[string]float64{"limit": 1e308, "windowMs": 1000}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1e308, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 5e-324}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1e308}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1e308, Key: "alice"}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: strings.Repeat("x", 129)}}},
		{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{{AtMs: 0, Cost: 1, Key: string([]byte{0xff})}}},
	}
	for _, language := range []string{LanguageGo, LanguagePython, LanguageJava, LanguageJavaScript} {
		runner, err := registry.Runner(language)
		if err != nil {
			t.Fatal(err)
		}
		for index, request := range invalid {
			if _, err := runner.Run(context.Background(), request); err == nil {
				t.Errorf("%s invalid fixture %d returned nil error", language, index)
			}
		}
	}
}

func assertContinuousDecisionTrace(t *testing.T, language string, events []TraceEvent) {
	t.Helper()
	for index := 0; index+1 < len(events); index++ {
		first, second := events[index], events[index+1]
		isPair := (first.StepID == "fixed.locate-window" && second.StepID == "fixed.decision") ||
			(first.StepID == "sliding-log.evict" && second.StepID == "sliding-log.decision") ||
			(first.StepID == "sliding-counter.rotate" && second.StepID == "sliding-counter.estimate") ||
			(first.StepID == "sliding-counter.estimate" && second.StepID == "sliding-counter.decision") ||
			(first.StepID == "token.refill" && second.StepID == "token.decision") ||
			(first.StepID == "leaky.drain" && second.StepID == "leaky.decision")
		if isPair && !reflect.DeepEqual(first.After, second.Before) {
			t.Errorf("%s trace jumps between %s and %s: after=%v before=%v", language, first.StepID, second.StepID, first.After, second.Before)
		}
	}
}

func assertTraceEventParity(t *testing.T, language string, index int, want, got TraceEvent) {
	t.Helper()
	if got.Seq != want.Seq || got.StepID != want.StepID || got.Actor != want.Actor ||
		got.TimestampMs != want.TimestampMs || got.Reason != want.Reason {
		t.Fatalf("%s event[%d] metadata = %+v, Go = %+v", language, index, got, want)
	}
	if !reflect.DeepEqual(wireValue(t, got.Before), wireValue(t, want.Before)) {
		t.Errorf("%s event[%d].before = %#v, Go = %#v", language, index, got.Before, want.Before)
	}
	if !reflect.DeepEqual(wireValue(t, got.After), wireValue(t, want.After)) {
		t.Errorf("%s event[%d].after = %#v, Go = %#v", language, index, got.After, want.After)
	}
	if (got.Decision == nil) != (want.Decision == nil) {
		t.Errorf("%s event[%d].decision presence = %v, Go = %v", language, index, got.Decision != nil, want.Decision != nil)
	} else if got.Decision != nil {
		assertDecisionParity(t, language, []Decision{*want.Decision}, []Decision{*got.Decision})
	}
}

func wireValue(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		t.Fatal(err)
	}
	return normalized
}

func assertDecisionParity(t *testing.T, language string, want, got []Decision) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s returned %d decisions, want %d", language, len(got), len(want))
	}
	for index := range want {
		if got[index].Allowed != want[index].Allowed || got[index].Reason != want[index].Reason ||
			math.Abs(got[index].Remaining-want[index].Remaining) > 1e-6 ||
			math.Abs(got[index].RetryAfterMs-want[index].RetryAfterMs) > 1e-6 ||
			math.Abs(got[index].ResetAtMs-want[index].ResetAtMs) > 1e-6 {
			t.Errorf("%s decision[%d] = %+v, Go = %+v", language, index, got[index], want[index])
		}
	}
}
