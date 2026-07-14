package server

import (
	"reflect"
	"testing"
)

func TestDefaultCatalogProvidesCompleteScenarioBriefs(t *testing.T) {
	catalog := DefaultCatalog()
	if len(catalog.Scenarios) != 14 {
		t.Fatalf("catalog has %d scenarios, want 14", len(catalog.Scenarios))
	}

	validAdmission := map[string]bool{"allow": true, "deny": true}
	validCaseKind := map[string]bool{"allow": true, "deny": true, "observe": true}
	seen := make(map[string]bool, len(catalog.Scenarios))
	for _, scenario := range catalog.Scenarios {
		if seen[scenario.ID] {
			t.Errorf("duplicate scenario id %q", scenario.ID)
		}
		seen[scenario.ID] = true
		if scenario.Brief == nil {
			t.Errorf("scenario %q has no brief", scenario.ID)
			continue
		}
		brief := scenario.Brief
		if brief.Traffic == "" || brief.Expected.Summary == "" {
			t.Errorf("scenario %q has incomplete brief: %+v", scenario.ID, brief)
		}
		if scenario.ID == "local-vs-shared" && !brief.ReplicaScoped {
			t.Errorf("scenario %q must expose replica-scoped policy state", scenario.ID)
		}
		if scenario.ID != "local-vs-shared" && brief.ReplicaScoped {
			t.Errorf("scenario %q unexpectedly exposes replica-scoped policy state", scenario.ID)
		}
		if scenario.ID == "multi-region-quota" && !brief.Conceptual {
			t.Errorf("scenario %q must be marked conceptual", scenario.ID)
		}
		if scenario.ID != "multi-region-quota" && brief.Conceptual {
			t.Errorf("scenario %q unexpectedly marked conceptual", scenario.ID)
		}
		switch scenario.Tier {
		case "core":
			if len(brief.Expected.Admissions) != len(scenario.RequestTimeline) {
				t.Errorf("core scenario %q has %d admissions for %d requests", scenario.ID, len(brief.Expected.Admissions), len(scenario.RequestTimeline))
			}
			for index, admission := range brief.Expected.Admissions {
				if !validAdmission[admission] {
					t.Errorf("core scenario %q admission[%d] = %q, want allow or deny", scenario.ID, index, admission)
				}
			}
			if len(brief.Expected.Cases) != 0 {
				t.Errorf("core scenario %q unexpectedly has conditional cases: %+v", scenario.ID, brief.Expected.Cases)
			}
		case "system":
			if len(brief.Expected.Admissions) != 0 {
				t.Errorf("system scenario %q unexpectedly has canonical admissions: %v", scenario.ID, brief.Expected.Admissions)
			}
			if len(brief.Expected.Cases) == 0 {
				t.Errorf("system scenario %q has no expected cases", scenario.ID)
			}
			for index, expectedCase := range brief.Expected.Cases {
				if expectedCase.When == "" || expectedCase.Result == "" || !validCaseKind[expectedCase.Kind] {
					t.Errorf("system scenario %q case[%d] is invalid: %+v", scenario.ID, index, expectedCase)
				}
			}
		default:
			t.Errorf("scenario %q has unexpected tier %q", scenario.ID, scenario.Tier)
		}
	}
}

func TestCoreScenarioBriefAdmissionsMatchLearningBaseline(t *testing.T) {
	want := map[string][]string{
		"steady-traffic":      {"allow", "allow", "allow", "allow"},
		"burst-capacity":      {"allow", "allow", "allow", "deny", "allow"},
		"window-boundary":     {"allow", "allow", "allow", "allow", "allow", "allow"},
		"exactness-vs-memory": {"allow", "allow", "allow", "deny", "allow"},
		"smoothed-output":     {"allow", "allow", "allow", "deny", "allow", "allow"},
		"concurrent-callers":  {"allow", "allow", "allow", "allow", "allow", "allow"},
		"clock-input-safety":  {"allow", "allow", "allow", "allow"},
	}

	for _, scenario := range DefaultCatalog().Scenarios {
		if scenario.Tier != "core" {
			continue
		}
		if scenario.Brief == nil {
			t.Errorf("core scenario %q has no brief", scenario.ID)
			continue
		}
		if !reflect.DeepEqual(scenario.Brief.Expected.Admissions, want[scenario.ID]) {
			t.Errorf("scenario %q admissions = %v, want %v", scenario.ID, scenario.Brief.Expected.Admissions, want[scenario.ID])
		}
		delete(want, scenario.ID)
	}
	if len(want) != 0 {
		t.Errorf("catalog is missing core scenarios: %v", want)
	}
}
