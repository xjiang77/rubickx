package mediator

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Mediator struct{ routes map[string][]string }

func (m Mediator) Dispatch(event map[string]any) ([]map[string]any, error) {
	from := event["from"].(string)
	recipients, ok := m.routes[from]
	if !ok {
		return nil, &PatternError{code: "unsupported_sender"}
	}
	out := make([]map[string]any, 0, len(recipients))
	for _, target := range recipients {
		out = append(out, map[string]any{"from": from, "to": target, "message": event["message"]})
	}
	return out, nil
}
func Evaluate(input map[string]any) (any, error) {
	m := Mediator{map[string][]string{"customer": {"agent", "bot"}, "agent": {"customer"}, "bot": {"agent"}}}
	deliveries := []map[string]any{}
	raw, _ := input["events"].([]any)
	for _, event := range raw {
		values, err := m.Dispatch(event.(map[string]any))
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, values...)
	}
	return map[string]any{"deliveries": deliveries}, nil
}
