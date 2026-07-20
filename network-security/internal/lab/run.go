package lab

import (
	"context"
	"fmt"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
)

type runner func(context.Context) (evidence.LabResult, error)

var runners = map[string]runner{
	"LAB-NETSEC-01": runLab01,
	"LAB-NETSEC-02": runLab02,
	"LAB-NETSEC-03": runLab03,
	"LAB-NETSEC-04": runLab04,
	"LAB-NETSEC-05": runLab05,
	"LAB-NETSEC-06": runLab06,
	"LAB-NETSEC-07": runLab07,
	"LAB-NETSEC-08": runLab08,
	"LAB-NETSEC-09": runLab09,
	"LAB-NETSEC-10": runLab10,
}

func IDs() []string {
	ids := make([]string, 0, 10)
	for index := 1; index <= 10; index++ {
		ids = append(ids, fmt.Sprintf("LAB-NETSEC-%02d", index))
	}
	return ids
}

func Run(ctx context.Context, id string) (evidence.LabResult, error) {
	run, ok := runners[id]
	if !ok {
		return evidence.LabResult{}, fmt.Errorf("unknown lab %q", id)
	}
	return run(ctx)
}

func RunAll(ctx context.Context) (evidence.Report, error) {
	report := evidence.Report{SchemaVersion: 2}
	for _, id := range IDs() {
		result, err := Run(ctx, id)
		if err != nil {
			return evidence.Report{}, fmt.Errorf("%s: %w", id, err)
		}
		report.Labs = append(report.Labs, result)
	}
	return report, nil
}

func event(id, outcome, decision, reason string) evidence.Event {
	return evidence.NewEvent(id, outcome, decision, reason)
}

func scenarioEvent(
	id, outcome, scenarioID, stage, component, evidenceKind, observedState, decision, reason string,
) evidence.Event {
	result := event(id, outcome, decision, reason)
	result.ScenarioID = scenarioID
	result.Stage = stage
	result.Component = component
	result.EvidenceKind = evidenceKind
	result.ObservedState = observedState
	return result
}
