package strategy

import "sort"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Strategy interface {
	Select([]map[string]any) (string, error)
}
type metricStrategy struct{ field string }

func (s metricStrategy) Select(candidates []map[string]any) (string, error) {
	healthy := make([]map[string]any, 0)
	for _, value := range candidates {
		if value["healthy"] == true {
			healthy = append(healthy, value)
		}
	}
	if len(healthy) == 0 {
		return "", &PatternError{code: "no_healthy_candidate"}
	}
	sort.Slice(healthy, func(i, j int) bool {
		a, b := healthy[i][s.field].(float64), healthy[j][s.field].(float64)
		if a == b {
			return healthy[i]["name"].(string) < healthy[j]["name"].(string)
		}
		return a < b
	})
	return healthy[0]["name"].(string), nil
}
func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["selections"].([]any)
	selected := make([]string, 0, len(raw))
	for _, item := range raw {
		value := item.(map[string]any)
		var current Strategy
		switch value["strategy"] {
		case "cost":
			current = metricStrategy{"cost"}
		case "latency":
			current = metricStrategy{"latency"}
		default:
			return nil, &PatternError{code: "unsupported_strategy"}
		}
		rawCandidates, _ := value["candidates"].([]any)
		candidates := make([]map[string]any, 0, len(rawCandidates))
		for _, candidate := range rawCandidates {
			candidates = append(candidates, candidate.(map[string]any))
		}
		result, err := current.Select(candidates)
		if err != nil {
			return nil, err
		}
		selected = append(selected, result)
	}
	return map[string]any{"selected": selected}, nil
}
