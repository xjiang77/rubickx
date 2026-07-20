package timeout

type FakeClock struct{ now int }

func (c *FakeClock) Advance(v int) { c.now += v }
func Evaluate(input map[string]any) (any, error) {
	clock := &FakeClock{}
	deadline := int(input["deadline_ms"].(float64))
	outcomes := []map[string]any{}
	raw, _ := input["operations"].([]any)
	for _, item := range raw {
		op := item.(map[string]any)
		duration := int(op["duration_ms"].(float64))
		if clock.now+duration <= deadline {
			clock.Advance(duration)
			outcomes = append(outcomes, map[string]any{"name": op["name"], "status": "completed", "outcome": "success"})
		} else {
			clock.Advance(max(0, deadline-clock.now))
			outcome := "abandoned"
			if op["side_effecting"] == true {
				outcome = "unknown"
			}
			outcomes = append(outcomes, map[string]any{"name": op["name"], "status": "timed_out", "outcome": outcome})
			break
		}
	}
	return map[string]any{"outcomes": outcomes, "now_ms": clock.now}, nil
}
