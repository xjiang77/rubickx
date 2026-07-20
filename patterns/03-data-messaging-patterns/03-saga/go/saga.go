package saga

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func names(steps []map[string]any) []string {
	result := make([]string, 0, len(steps))
	for _, step := range steps {
		result = append(result, step["name"].(string))
	}
	return result
}

func Evaluate(input map[string]any) (any, error) {
	completed := []map[string]any{}
	actions := []string{}
	raw, _ := input["steps"].([]any)
	for _, item := range raw {
		step := item.(map[string]any)
		name := step["name"].(string)
		actions = append(actions, "execute:"+name)
		result := step["result"].(string)
		if result == "success" {
			completed = append(completed, step)
			continue
		}
		if result != "failure" {
			return nil, &PatternError{code: "unknown_step_result"}
		}
		actions = append(actions, "failed:"+name)
		failed := []string{}
		for i := len(completed) - 1; i >= 0; i-- {
			prior := completed[i]
			outcome := "success"
			if value, ok := prior["compensation"]; ok {
				outcome = value.(string)
			}
			if outcome != "success" && outcome != "failure" {
				return nil, &PatternError{code: "unknown_compensation_result"}
			}
			priorName := prior["name"].(string)
			actions = append(actions, "compensate:"+priorName+":"+outcome)
			if outcome == "failure" {
				failed = append(failed, priorName)
			}
		}
		status := "compensated"
		if len(failed) > 0 {
			status = "recovery_required"
		}
		return map[string]any{"status": status, "completed": names(completed), "failed_step": name, "failed_compensations": failed, "actions": actions}, nil
	}
	return map[string]any{"status": "completed", "completed": names(completed), "failed_step": "none", "failed_compensations": []string{}, "actions": actions}, nil
}
