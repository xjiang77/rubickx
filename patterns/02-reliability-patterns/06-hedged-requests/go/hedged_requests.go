package hedgedrequests

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type candidate struct {
	time int
	name string
}

func Evaluate(input map[string]any) (any, error) {
	if input["idempotent"] != true {
		return nil, &PatternError{code: "hedge_not_allowed"}
	}
	primary := input["primary"].(map[string]any)
	delay := int(input["hedge_delay_ms"].(float64))
	primaryLatency := int(primary["latency_ms"].(float64))
	if primary["result"] == "success" && primaryLatency <= delay {
		return map[string]any{"winner": "primary", "completed_ms": primaryLatency, "attempts": 1, "cancelled": []string{}}, nil
	}
	start := delay
	if primary["result"] == "failure" && primaryLatency < start {
		start = primaryLatency
	}
	hedge := input["hedge"].(map[string]any)
	values := []candidate{}
	if primary["result"] == "success" {
		values = append(values, candidate{primaryLatency, "primary"})
	}
	if hedge["result"] == "success" {
		values = append(values, candidate{start + int(hedge["latency_ms"].(float64)), "hedge"})
	}
	if len(values) == 0 {
		return nil, &PatternError{code: "all_attempts_failed"}
	}
	winner := values[0]
	for _, value := range values[1:] {
		if value.time < winner.time {
			winner = value
		}
	}
	cancelled := []string{}
	for _, value := range values {
		if value.name != winner.name && winner.time < value.time {
			cancelled = append(cancelled, value.name)
		}
	}
	return map[string]any{"winner": winner.name, "completed_ms": winner.time, "attempts": 2, "cancelled": cancelled}, nil
}
