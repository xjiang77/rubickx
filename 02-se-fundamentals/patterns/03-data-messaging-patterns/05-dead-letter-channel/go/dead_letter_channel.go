package deadletterchannel

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func Evaluate(input map[string]any) (any, error) {
	budget := int(input["max_attempts"].(float64))
	if budget < 1 {
		return nil, &PatternError{code: "invalid_attempt_budget"}
	}
	processed := []string{}
	dead := []map[string]any{}
	receipts := []string{}
	byID := map[string]map[string]any{}
	messages, _ := input["messages"].([]any)
	for _, item := range messages {
		message := item.(map[string]any)
		id := message["id"].(string)
		if _, ok := byID[id]; ok {
			return nil, &PatternError{code: "duplicate_message_id"}
		}
		byID[id] = message
		outcomes, _ := message["outcomes"].([]any)
		if len(outcomes) == 0 {
			return nil, &PatternError{code: "missing_outcome"}
		}
		for index := 0; index < budget; index++ {
			outcome := outcomes[len(outcomes)-1].(string)
			if index < len(outcomes) {
				outcome = outcomes[index].(string)
			}
			attempt := index + 1
			switch outcome {
			case "success":
				processed = append(processed, id)
				receipts = append(receipts, "processed:"+id+":"+itoa(attempt))
				index = budget
			case "permanent":
				dead = append(dead, map[string]any{"id": id, "reason": "permanent", "attempts": attempt, "status": "dead"})
				receipts = append(receipts, "dead:"+id+":permanent:"+itoa(attempt))
				index = budget
			case "transient":
				if attempt == budget {
					dead = append(dead, map[string]any{"id": id, "reason": "transient_exhausted", "attempts": attempt, "status": "dead"})
					receipts = append(receipts, "dead:"+id+":transient_exhausted:"+itoa(attempt))
				} else {
					receipts = append(receipts, "retry:"+id+":"+itoa(attempt))
				}
			default:
				return nil, &PatternError{code: "unknown_processing_outcome"}
			}
		}
	}
	replay, _ := input["replay_ids"].([]any)
	for _, item := range replay {
		id := item.(string)
		found := -1
		for index := range dead {
			if dead[index]["id"] == id && dead[index]["status"] == "dead" {
				found = index
				break
			}
		}
		if found < 0 {
			return nil, &PatternError{code: "replay_not_dead"}
		}
		outcome := "failure"
		if value, ok := byID[id]["replay_outcome"]; ok {
			outcome = value.(string)
		}
		if outcome == "success" {
			dead[found]["status"] = "replayed"
			processed = append(processed, id)
			receipts = append(receipts, "replayed:"+id)
		} else if outcome == "failure" {
			receipts = append(receipts, "replay_failed:"+id)
		} else {
			return nil, &PatternError{code: "unknown_replay_outcome"}
		}
	}
	return map[string]any{"processed": processed, "dead_letters": dead, "receipts": receipts}, nil
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
