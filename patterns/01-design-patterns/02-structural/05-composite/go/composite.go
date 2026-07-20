package composite

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Component interface {
	Evaluate(map[string]any, *[]string) bool
}
type equals struct {
	field string
	value any
}

func (e equals) Evaluate(context map[string]any, trace *[]string) bool {
	*trace = append(*trace, e.field)
	return context[e.field] == e.value
}

type all struct{ children []Component }

func (a all) Evaluate(context map[string]any, trace *[]string) bool {
	for _, child := range a.children {
		if !child.Evaluate(context, trace) {
			return false
		}
	}
	return true
}

type anyOf struct{ children []Component }

func (a anyOf) Evaluate(context map[string]any, trace *[]string) bool {
	for _, child := range a.children {
		if child.Evaluate(context, trace) {
			return true
		}
	}
	return false
}
func build(value map[string]any) (Component, error) {
	kind, _ := value["type"].(string)
	if kind == "equals" {
		return equals{value["field"].(string), value["value"]}, nil
	}
	if kind == "all" || kind == "any" {
		raw, _ := value["children"].([]any)
		children := make([]Component, 0, len(raw))
		for _, item := range raw {
			child, err := build(item.(map[string]any))
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		}
		if kind == "all" {
			return all{children}, nil
		}
		return anyOf{children}, nil
	}
	return nil, &PatternError{code: "unsupported_node"}
}
func Evaluate(input map[string]any) (any, error) {
	root, err := build(input["tree"].(map[string]any))
	if err != nil {
		return nil, err
	}
	context, _ := input["context"].(map[string]any)
	trace := []string{}
	allowed := root.Evaluate(context, &trace)
	return map[string]any{"allowed": allowed, "evaluated": trace}, nil
}
