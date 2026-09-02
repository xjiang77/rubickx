package publishersubscriber

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func subscribes(subscriber map[string]any, topic string) bool {
	raw, _ := subscriber["topics"].([]any)
	for _, item := range raw {
		if item.(string) == topic {
			return true
		}
	}
	return false
}

func Evaluate(input map[string]any) (any, error) {
	subscribers, _ := input["subscribers"].([]any)
	seen := map[string]bool{}
	for _, item := range subscribers {
		name := item.(map[string]any)["name"].(string)
		if seen[name] {
			return nil, &PatternError{code: "duplicate_subscriber"}
		}
		seen[name] = true
	}
	events, _ := input["events"].([]any)
	deliveries := []map[string]any{}
	for _, eventItem := range events {
		event := eventItem.(map[string]any)
		for _, subscriberItem := range subscribers {
			subscriber := subscriberItem.(map[string]any)
			if !subscribes(subscriber, event["topic"].(string)) {
				continue
			}
			outcome := "success"
			if value, ok := subscriber["outcome"]; ok {
				outcome = value.(string)
			}
			if outcome != "success" && outcome != "failure" {
				return nil, &PatternError{code: "unknown_delivery_outcome"}
			}
			status := "delivered"
			if outcome == "failure" {
				status = "failed"
			}
			deliveries = append(deliveries, map[string]any{"event_id": event["id"], "subscriber": subscriber["name"], "status": status})
		}
	}
	return map[string]any{"published": len(events), "deliveries": deliveries}, nil
}
