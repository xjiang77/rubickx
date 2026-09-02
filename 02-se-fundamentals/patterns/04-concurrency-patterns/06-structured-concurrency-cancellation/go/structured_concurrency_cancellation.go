package structuredconcurrencycancellation

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }
func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["children"].([]any)
	orderRaw, _ := input["completion_order"].([]any)
	names := []string{}
	children := map[string]map[string]any{}
	states := map[string]string{}
	for _, item := range raw {
		child := item.(map[string]any)
		name := child["name"].(string)
		if _, ok := children[name]; ok {
			return nil, &PatternError{code: "duplicate_child"}
		}
		names = append(names, name)
		children[name] = child
		states[name] = "pending"
	}
	order := []string{}
	seen := map[string]bool{}
	for _, item := range orderRaw {
		name := item.(string)
		if _, ok := children[name]; !ok || seen[name] {
			return nil, &PatternError{code: "invalid_completion_order"}
		}
		seen[name] = true
		order = append(order, name)
	}
	if len(order) != len(names) {
		return nil, &PatternError{code: "invalid_completion_order"}
	}
	results := []string{}
	status := "completed"
	completed := 0
	cancelAfter := -1
	if value, ok := input["parent_cancel_after"].(float64); ok {
		cancelAfter = int(value)
	}
	for _, name := range order {
		if cancelAfter >= 0 && completed >= cancelAfter {
			status = "cancelled"
			break
		}
		child := children[name]
		switch child["outcome"].(string) {
		case "success":
			states[name] = "completed"
			results = append(results, name+":"+child["value"].(string))
			completed++
		case "failure":
			states[name] = "failed"
			status = "failed"
		default:
			return nil, &PatternError{code: "unknown_outcome"}
		}
		if status == "failed" {
			break
		}
	}
	if status != "completed" {
		for _, name := range names {
			if states[name] == "pending" {
				states[name] = "cancelled"
			}
		}
	}
	childStates := []string{}
	for _, name := range names {
		childStates = append(childStates, name+":"+states[name])
	}
	return map[string]any{"parent_status": status, "child_states": childStates, "results": results, "joined_count": len(names), "leaked": 0}, nil
}
