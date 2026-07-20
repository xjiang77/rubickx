package fanoutfanin

import (
	"sort"
)

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type task struct {
	id, outcome, value string
	at, index          int
}

func strings(items []task) []string {
	out := []string{}
	for _, item := range items {
		out = append(out, item.id)
	}
	return out
}
func Evaluate(input map[string]any) (any, error) {
	mode, _ := input["mode"].(string)
	if mode != "all" && mode != "first_success" {
		return nil, &PatternError{code: "unknown_mode"}
	}
	if mode == "first_success" {
		if side, ok := input["side_effecting"].(bool); ok && side {
			return nil, &PatternError{code: "fanout_not_allowed"}
		}
	}
	raw, _ := input["tasks"].([]any)
	tasks := []task{}
	seen := map[string]bool{}
	for index, item := range raw {
		value := item.(map[string]any)
		id := value["id"].(string)
		if seen[id] {
			return nil, &PatternError{code: "duplicate_task"}
		}
		seen[id] = true
		tasks = append(tasks, task{id: id, outcome: value["outcome"].(string), value: value["value"].(string), at: int(value["complete_at"].(float64)), index: index})
	}
	sort.SliceStable(tasks, func(i, j int) bool { return tasks[i].at < tasks[j].at })
	completion := []string{}
	results := []string{}
	failures := []string{}
	for position, item := range tasks {
		completion = append(completion, item.id)
		switch item.outcome {
		case "success":
			results = append(results, item.id+":"+item.value)
			if mode == "first_success" {
				return map[string]any{"completion_order": completion, "results": results, "failures": failures, "cancelled": strings(tasks[position+1:]), "status": "completed"}, nil
			}
		case "failure":
			failures = append(failures, item.id)
		default:
			return nil, &PatternError{code: "unknown_outcome"}
		}
	}
	if mode == "first_success" {
		return nil, &PatternError{code: "all_tasks_failed"}
	}
	status := "completed"
	if len(failures) > 0 {
		status = "partial"
	}
	return map[string]any{"completion_order": completion, "results": results, "failures": failures, "cancelled": []string{}, "status": status}, nil
}
