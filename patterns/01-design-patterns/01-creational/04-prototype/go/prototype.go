package prototype

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type RoutingPolicy struct {
	Name, Primary string
	Fallbacks     []string
}

func (p RoutingPolicy) Clone() RoutingPolicy {
	fallbacks := append(make([]string, 0, len(p.Fallbacks)), p.Fallbacks...)
	return RoutingPolicy{Name: p.Name + "-copy", Primary: p.Primary, Fallbacks: fallbacks}
}
func (p RoutingPolicy) Value() map[string]any {
	return map[string]any{"name": p.Name, "primary": p.Primary, "fallbacks": p.Fallbacks}
}
func Evaluate(input map[string]any) (any, error) {
	b := input["base"].(map[string]any)
	name, _ := b["name"].(string)
	primary, _ := b["primary"].(string)
	if name == "" || primary == "" {
		return nil, &PatternError{code: "invalid_prototype"}
	}
	fallbacks := make([]string, 0, len(b["fallbacks"].([]any)))
	for _, v := range b["fallbacks"].([]any) {
		fallbacks = append(fallbacks, v.(string))
	}
	original := RoutingPolicy{name, primary, fallbacks}
	clone := original.Clone()
	if v, ok := input["override_primary"].(string); ok {
		clone.Primary = v
	}
	if v, ok := input["append_fallback"].(string); ok {
		clone.Fallbacks = append(clone.Fallbacks, v)
	}
	return map[string]any{"original": original.Value(), "clone": clone.Value()}, nil
}
