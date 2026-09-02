package pipeline

import "strconv"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }
func ints(raw []any) []int {
	result := []int{}
	for _, item := range raw {
		result = append(result, int(item.(float64)))
	}
	return result
}
func Evaluate(input map[string]any) (any, error) {
	rawItems, _ := input["items"].([]any)
	items := ints(rawItems)
	rawStages, _ := input["stages"].([]any)
	outputs := []int{}
	receipts := []string{}
	limit := -1
	if value, ok := input["consumer_limit"].(float64); ok {
		limit = int(value)
	}
	for index, item := range items {
		if limit >= 0 && len(outputs) >= limit {
			return map[string]any{"outputs": outputs, "status": "cancelled", "failed_stage": "none", "cancelled_items": items[index:], "stage_receipts": receipts, "upstream_cancelled": true}, nil
		}
		value := item
		for _, raw := range rawStages {
			stage := raw.(string)
			before := value
			switch stage {
			case "reject_negative":
				if value < 0 {
					return map[string]any{"outputs": outputs, "status": "failed", "failed_stage": stage, "cancelled_items": items[index+1:], "stage_receipts": append(receipts, "failed:"+stage+":"+strconv.Itoa(value)), "upstream_cancelled": true}, nil
				}
			case "double":
				value *= 2
			case "increment":
				value++
			default:
				return nil, &PatternError{code: "unknown_stage"}
			}
			receipts = append(receipts, stage+":"+strconv.Itoa(before)+"->"+strconv.Itoa(value))
		}
		outputs = append(outputs, value)
	}
	return map[string]any{"outputs": outputs, "status": "completed", "failed_stage": "none", "cancelled_items": []int{}, "stage_receipts": receipts, "upstream_cancelled": false}, nil
}
