package boundedproducerconsumer

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func Evaluate(input map[string]any) (any, error) {
	capacity := int(input["capacity"].(float64))
	if capacity < 0 {
		return nil, &PatternError{code: "invalid_capacity"}
	}
	queue := []string{}
	consumed := []string{}
	receipts := []string{}
	closed := false
	actions, _ := input["actions"].([]any)
	for _, item := range actions {
		action := item.(map[string]any)
		operation := action["op"].(string)
		switch operation {
		case "produce":
			value := action["item"].(string)
			if closed {
				return nil, &PatternError{code: "produce_after_close"}
			}
			if len(queue) >= capacity {
				receipts = append(receipts, "backpressured:"+value)
			} else {
				queue = append(queue, value)
				receipts = append(receipts, "queued:"+value)
			}
		case "consume":
			if len(queue) > 0 {
				value := queue[0]
				queue = queue[1:]
				consumed = append(consumed, value)
				receipts = append(receipts, "consumed:"+value)
			} else if closed {
				receipts = append(receipts, "end")
			} else {
				receipts = append(receipts, "empty")
			}
		case "close":
			if closed {
				return nil, &PatternError{code: "already_closed"}
			}
			closed = true
			receipts = append(receipts, "closed")
		default:
			return nil, &PatternError{code: "unknown_action"}
		}
	}
	return map[string]any{"receipts": receipts, "consumed": consumed, "remaining": queue, "closed": closed}, nil
}
