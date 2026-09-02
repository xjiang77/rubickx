package retry

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Scheduler struct{ delays []int }

func (s *Scheduler) Wait(value int) { s.delays = append(s.delays, value) }
func Evaluate(input map[string]any) (any, error) {
	scheduler := &Scheduler{delays: []int{}}
	raw, _ := input["outcomes"].([]any)
	maxAttempts := int(input["max_attempts"].(float64))
	base := int(input["base_delay_ms"].(float64))
	attempts := 0
	for attempts < maxAttempts {
		outcome := "transient"
		if attempts < len(raw) {
			outcome = raw[attempts].(string)
		}
		attempts++
		switch outcome {
		case "success":
			return map[string]any{"status": "success", "attempts": attempts, "delays_ms": scheduler.delays}, nil
		case "permanent":
			return nil, &PatternError{code: "non_retryable"}
		case "transient":
		default:
			return nil, &PatternError{code: "unknown_outcome"}
		}
		if attempts < maxAttempts {
			scheduler.Wait(base << (attempts - 1))
		}
	}
	return map[string]any{"status": "exhausted", "attempts": attempts, "delays_ms": scheduler.delays}, nil
}
