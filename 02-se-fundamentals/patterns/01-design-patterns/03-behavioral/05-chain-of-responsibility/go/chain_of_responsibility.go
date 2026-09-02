package chainofresponsibility

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Handler interface {
	Handle(map[string]any, *[]string) (string, error)
}
type auth struct{ next Handler }

func (h auth) Handle(r map[string]any, v *[]string) (string, error) {
	*v = append(*v, "auth")
	if r["token"] != "valid" {
		return "", &PatternError{code: "unauthenticated"}
	}
	return h.next.Handle(r, v)
}

type quota struct{ next Handler }

func (h quota) Handle(r map[string]any, v *[]string) (string, error) {
	*v = append(*v, "quota")
	if r["quota"].(float64) <= 0 {
		return "", &PatternError{code: "quota_exhausted"}
	}
	return h.next.Handle(r, v)
}

type execute struct{}

func (execute) Handle(r map[string]any, v *[]string) (string, error) {
	*v = append(*v, "execute")
	return "accepted:" + r["payload"].(string), nil
}
func Evaluate(input map[string]any) (any, error) {
	var chain Handler = auth{quota{execute{}}}
	raw, _ := input["requests"].([]any)
	responses := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		visited := []string{}
		decision, err := chain.Handle(item.(map[string]any), &visited)
		if err != nil {
			return nil, err
		}
		responses = append(responses, map[string]any{"decision": decision, "visited": visited})
	}
	return map[string]any{"responses": responses}, nil
}
