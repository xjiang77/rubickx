package transactionaloutbox

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["relay_attempts"].([]any)
	if input["commit"] != true {
		if len(raw) > 0 {
			return nil, &PatternError{code: "relay_without_commit"}
		}
		return map[string]any{"aggregate_state": "absent", "outbox_status": "absent", "deliveries": []string{}, "relay_receipts": []string{}}, nil
	}
	messageID := input["message_id"].(string)
	status := "pending"
	deliveries := []string{}
	receipts := []string{}
	for _, item := range raw {
		attempt := item.(string)
		if status == "sent" {
			receipts = append(receipts, "skipped_already_sent:"+messageID)
			continue
		}
		switch attempt {
		case "crash_before_publish":
			receipts = append(receipts, "crash_before_publish:"+messageID)
		case "crash_after_publish":
			deliveries = append(deliveries, messageID)
			receipts = append(receipts, "crash_after_publish:"+messageID)
		case "success":
			deliveries = append(deliveries, messageID)
			receipts = append(receipts, "published:"+messageID)
			status = "sent"
		default:
			return nil, &PatternError{code: "unknown_relay_outcome"}
		}
	}
	return map[string]any{"aggregate_state": input["new_state"], "outbox_status": status, "deliveries": deliveries, "relay_receipts": receipts}, nil
}
