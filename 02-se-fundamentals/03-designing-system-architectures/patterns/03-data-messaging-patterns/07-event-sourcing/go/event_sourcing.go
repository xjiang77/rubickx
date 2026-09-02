package eventsourcing

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func asInt(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case float64:
		return int(number), true
	default:
		return 0, false
	}
}

func applyEvent(balance int, event map[string]any) (int, error) {
	amount, ok := asInt(event["amount"])
	if !ok || amount <= 0 {
		return 0, &PatternError{code: "invalid_event_amount"}
	}
	switch event["type"].(string) {
	case "deposited":
		return balance + amount, nil
	case "withdrawn":
		if amount > balance {
			return 0, &PatternError{code: "invalid_event_history"}
		}
		return balance - amount, nil
	default:
		return 0, &PatternError{code: "unknown_event_type"}
	}
}

func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["events"].([]any)
	balance := 0
	for _, item := range raw {
		var err error
		balance, err = applyEvent(balance, item.(map[string]any))
		if err != nil {
			return nil, err
		}
	}
	appended := []map[string]any{}
	if command, ok := input["command"].(map[string]any); ok {
		if int(command["expected_version"].(float64)) != len(raw) {
			return nil, &PatternError{code: "version_conflict"}
		}
		amount := int(command["amount"].(float64))
		if amount <= 0 {
			return nil, &PatternError{code: "invalid_command_amount"}
		}
		eventType := ""
		switch command["type"].(string) {
		case "deposit":
			eventType = "deposited"
		case "withdraw":
			if amount > balance {
				return nil, &PatternError{code: "insufficient_funds"}
			}
			eventType = "withdrawn"
		default:
			return nil, &PatternError{code: "unknown_command_type"}
		}
		event := map[string]any{"type": eventType, "amount": amount, "version": len(raw) + 1}
		var err error
		balance, err = applyEvent(balance, event)
		if err != nil {
			return nil, err
		}
		appended = append(appended, event)
	}
	count := len(raw) + len(appended)
	return map[string]any{"balance": balance, "version": count, "history_count": count, "appended_events": appended}, nil
}
