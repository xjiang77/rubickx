package circuitbreaker

type Breaker struct {
	threshold, cooldown, failures, now, openedAt, forwarded int
	state                                                   string
}

func (b *Breaker) Call(result string) string {
	if b.state == "open" {
		if b.now-b.openedAt < b.cooldown {
			return "rejected"
		}
		b.state = "half_open"
	}
	b.forwarded++
	if b.state == "half_open" {
		if result == "success" {
			b.state = "closed"
			b.failures = 0
			return "probe_success"
		}
		b.state = "open"
		b.openedAt = b.now
		return "probe_failed"
	}
	if result == "success" {
		b.failures = 0
		return "success"
	}
	b.failures++
	if b.failures >= b.threshold {
		b.state = "open"
		b.openedAt = b.now
		return "opened"
	}
	return "failure"
}
func Evaluate(input map[string]any) (any, error) {
	b := &Breaker{threshold: int(input["threshold"].(float64)), cooldown: int(input["cooldown"].(float64)), state: "closed"}
	decisions := []string{}
	raw, _ := input["events"].([]any)
	for _, item := range raw {
		event := item.(map[string]any)
		if value, ok := event["advance"].(float64); ok {
			b.now += int(value)
			decisions = append(decisions, "advanced")
		} else {
			decisions = append(decisions, b.Call(event["result"].(string)))
		}
	}
	return map[string]any{"decisions": decisions, "final": b.state, "forwarded": b.forwarded}, nil
}
