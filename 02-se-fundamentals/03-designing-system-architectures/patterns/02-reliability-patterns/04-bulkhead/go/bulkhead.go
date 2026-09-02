package bulkhead

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }
func Evaluate(input map[string]any) (any, error) {
	capacities := map[string]int{}
	used := map[string]int{}
	for key, value := range input["capacities"].(map[string]any) {
		capacities[key] = int(value.(float64))
		used[key] = 0
	}
	decisions := []string{}
	raw, _ := input["actions"].([]any)
	for _, item := range raw {
		action := item.(map[string]any)
		pool := action["pool"].(string)
		capacity, ok := capacities[pool]
		if !ok {
			return nil, &PatternError{code: "unknown_pool"}
		}
		switch action["op"] {
		case "acquire":
			if used[pool] >= capacity {
				decisions = append(decisions, "rejected:"+pool)
			} else {
				used[pool]++
				decisions = append(decisions, "accepted:"+pool)
			}
		case "release":
			if used[pool] <= 0 {
				return nil, &PatternError{code: "over_release"}
			}
			used[pool]--
			decisions = append(decisions, "released:"+pool)
		default:
			return nil, &PatternError{code: "unsupported_action"}
		}
	}
	out := map[string]any{}
	for key, value := range used {
		out[key] = value
	}
	return map[string]any{"decisions": decisions, "used": out}, nil
}
