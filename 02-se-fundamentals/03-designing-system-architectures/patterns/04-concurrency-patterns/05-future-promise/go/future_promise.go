package futurepromise

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }
func Evaluate(input map[string]any) (any, error) {
	state, value, errorValue := "pending", "none", "none"
	observations := []string{}
	terminal := 0
	actions, _ := input["actions"].([]any)
	for _, raw := range actions {
		action := raw.(map[string]any)
		op := action["op"].(string)
		switch op {
		case "complete", "fail", "cancel":
			if state != "pending" {
				return nil, &PatternError{code: "already_completed"}
			}
			terminal++
			if op == "complete" {
				state = "fulfilled"
				value = action["value"].(string)
			} else if op == "fail" {
				state = "rejected"
				errorValue = action["code"].(string)
			} else {
				state = "cancelled"
			}
		case "observe":
			if state == "pending" {
				observations = append(observations, "pending")
			} else if state == "fulfilled" {
				observations = append(observations, "value:"+value)
			} else if state == "rejected" {
				observations = append(observations, "error:"+errorValue)
			} else {
				observations = append(observations, "cancelled")
			}
		default:
			return nil, &PatternError{code: "unknown_action"}
		}
	}
	return map[string]any{"state": state, "observations": observations, "value": value, "error": errorValue, "terminal_count": terminal}, nil
}
