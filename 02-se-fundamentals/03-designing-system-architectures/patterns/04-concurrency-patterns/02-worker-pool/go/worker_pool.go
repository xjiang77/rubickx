package workerpool

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func Evaluate(input map[string]any) (any, error) {
	count := int(input["worker_count"].(float64))
	if count < 1 {
		return nil, &PatternError{code: "invalid_worker_count"}
	}
	available := make([]int, count)
	used := map[int]bool{}
	executions := []map[string]any{}
	jobs, _ := input["jobs"].([]any)
	for _, item := range jobs {
		job := item.(map[string]any)
		duration := int(job["duration"].(float64))
		if duration < 1 {
			return nil, &PatternError{code: "invalid_duration"}
		}
		outcome := job["outcome"].(string)
		if outcome != "success" && outcome != "failure" {
			return nil, &PatternError{code: "unknown_job_outcome"}
		}
		worker := 0
		for index := 1; index < count; index++ {
			if available[index] < available[worker] {
				worker = index
			}
		}
		start := available[worker]
		finish := start + duration
		available[worker] = finish
		used[worker] = true
		status := "completed"
		if outcome == "failure" {
			status = "failed"
		}
		executions = append(executions, map[string]any{"id": job["id"], "worker": worker, "start": start, "finish": finish, "status": status})
	}
	makespan := 0
	for _, value := range available {
		if value > makespan {
			makespan = value
		}
	}
	return map[string]any{"executions": executions, "workers_used": len(used), "makespan": makespan, "joined": true}, nil
}
