package loadshedding

import "sort"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type candidate struct {
	index int
	value map[string]any
}

func Evaluate(input map[string]any) (any, error) {
	rank := map[string]int{"high": 0, "normal": 1, "low": 2}
	raw, _ := input["requests"].([]any)
	values := make([]candidate, 0, len(raw))
	for index, item := range raw {
		value := item.(map[string]any)
		if _, ok := rank[value["priority"].(string)]; !ok {
			return nil, &PatternError{code: "unknown_priority"}
		}
		values = append(values, candidate{index, value})
	}
	sort.SliceStable(values, func(i, j int) bool {
		return rank[values[i].value["priority"].(string)] < rank[values[j].value["priority"].(string)]
	})
	capacity := int(input["capacity"].(float64))
	if capacity < 0 {
		capacity = 0
	}
	if capacity > len(values) {
		capacity = len(values)
	}
	accepted := []string{}
	set := map[string]bool{}
	for _, value := range values[:capacity] {
		id := value.value["id"].(string)
		accepted = append(accepted, id)
		set[id] = true
	}
	shed := []string{}
	for _, item := range raw {
		id := item.(map[string]any)["id"].(string)
		if !set[id] {
			shed = append(shed, id)
		}
	}
	return map[string]any{"accepted": accepted, "shed": shed, "goodput": len(accepted)}, nil
}
