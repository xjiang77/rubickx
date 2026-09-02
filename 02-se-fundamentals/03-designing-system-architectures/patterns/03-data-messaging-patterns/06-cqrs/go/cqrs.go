package cqrs

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type event struct{ delta int }

func Evaluate(input map[string]any) (any, error) {
	events := []event{}
	seen := map[string]bool{}
	writeBalance := 0
	commands, _ := input["commands"].([]any)
	for _, item := range commands {
		command := item.(map[string]any)
		id := command["id"].(string)
		if seen[id] {
			return nil, &PatternError{code: "duplicate_command_id"}
		}
		seen[id] = true
		delta := int(command["delta"].(float64))
		writeBalance += delta
		events = append(events, event{delta: delta})
	}
	projectionBalance, projectionVersion := 0, 0
	snapshots := []map[string]any{}
	targets, _ := input["projection_targets"].([]any)
	for _, item := range targets {
		target := int(item.(float64))
		if target < projectionVersion {
			return nil, &PatternError{code: "projection_regression"}
		}
		if target > len(events) {
			return nil, &PatternError{code: "projection_ahead"}
		}
		for _, event := range events[projectionVersion:target] {
			projectionBalance += event.delta
		}
		projectionVersion = target
		snapshots = append(snapshots, map[string]any{"balance": projectionBalance, "version": projectionVersion, "lag": len(events) - projectionVersion})
	}
	return map[string]any{"write_model": map[string]any{"balance": writeBalance, "version": len(events)}, "projection_snapshots": snapshots}, nil
}
