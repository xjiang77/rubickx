package idempotentconsumerinbox

import "sort"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

func Evaluate(input map[string]any) (any, error) {
	inbox := map[string]int{}
	receipts := []string{}
	balance := 0
	raw, _ := input["messages"].([]any)
	for _, item := range raw {
		message := item.(map[string]any)
		id := message["id"].(string)
		amount := int(message["amount"].(float64))
		if existing, ok := inbox[id]; ok {
			if existing != amount {
				return nil, &PatternError{code: "message_identity_conflict"}
			}
			receipts = append(receipts, "duplicate:"+id)
		} else {
			inbox[id] = amount
			balance += amount
			receipts = append(receipts, "applied:"+id)
		}
	}
	ids := make([]string, 0, len(inbox))
	for id := range inbox {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return map[string]any{"receipts": receipts, "balance": balance, "inbox_ids": ids}, nil
}
