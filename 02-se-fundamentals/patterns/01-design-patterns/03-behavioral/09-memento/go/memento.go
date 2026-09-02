package memento

import "fmt"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Memento struct {
	primary   string
	fallbacks []string
}
type RouteConfig struct {
	primary   string
	fallbacks []string
}

func (c *RouteConfig) Snapshot() Memento {
	return Memento{c.primary, append([]string{}, c.fallbacks...)}
}
func (c *RouteConfig) Restore(m Memento) {
	c.primary = m.primary
	c.fallbacks = append([]string{}, m.fallbacks...)
}
func (c *RouteConfig) Value() map[string]any {
	return map[string]any{"primary": c.primary, "fallbacks": append([]string{}, c.fallbacks...)}
}
func Evaluate(input map[string]any) (any, error) {
	initial := input["initial"].(map[string]any)
	fallbacks := []string{}
	for _, v := range initial["fallbacks"].([]any) {
		fallbacks = append(fallbacks, v.(string))
	}
	config := &RouteConfig{initial["primary"].(string), fallbacks}
	snapshots := []Memento{}
	audit := []string{}
	raw, _ := input["operations"].([]any)
	for _, item := range raw {
		op := item.(map[string]any)
		switch op["op"] {
		case "snapshot":
			snapshots = append(snapshots, config.Snapshot())
			audit = append(audit, fmt.Sprintf("snapshot:%d", len(snapshots)-1))
		case "set_primary":
			config.primary = op["value"].(string)
			audit = append(audit, "set_primary:"+config.primary)
		case "append_fallback":
			value := op["value"].(string)
			config.fallbacks = append(config.fallbacks, value)
			audit = append(audit, "append_fallback:"+value)
		case "restore":
			index := int(op["index"].(float64))
			if index < 0 || index >= len(snapshots) {
				return nil, &PatternError{code: "unknown_memento"}
			}
			config.Restore(snapshots[index])
			audit = append(audit, fmt.Sprintf("restore:%d", index))
		default:
			return nil, &PatternError{code: "unsupported_operation"}
		}
	}
	return map[string]any{"config": config.Value(), "snapshots": len(snapshots), "audit": audit}, nil
}
