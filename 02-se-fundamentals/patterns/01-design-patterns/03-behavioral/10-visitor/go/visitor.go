package visitor

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Node interface{ Accept(Visitor) any }
type Equals struct {
	field string
	value any
}

func (n Equals) Accept(v Visitor) any { return v.VisitEquals(n) }

type And struct{ children []Node }

func (n And) Accept(v Visitor) any { return v.VisitAnd(n) }

type Visitor interface {
	VisitEquals(Equals) any
	VisitAnd(And) any
}
type EvalVisitor struct{ context map[string]any }

func (v EvalVisitor) VisitEquals(n Equals) any { return v.context[n.field] == n.value }
func (v EvalVisitor) VisitAnd(n And) any {
	for _, child := range n.children {
		if !child.Accept(v).(bool) {
			return false
		}
	}
	return true
}

type FieldVisitor struct{}

func (FieldVisitor) VisitEquals(n Equals) any { return []string{n.field} }
func (v FieldVisitor) VisitAnd(n And) any {
	out := []string{}
	for _, child := range n.children {
		out = append(out, child.Accept(v).([]string)...)
	}
	return out
}
func buildVisitor(value map[string]any) (Node, error) {
	switch value["type"] {
	case "equals":
		return Equals{value["field"].(string), value["value"]}, nil
	case "and":
		raw, _ := value["children"].([]any)
		children := make([]Node, 0, len(raw))
		for _, item := range raw {
			child, err := buildVisitor(item.(map[string]any))
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		}
		return And{children}, nil
	default:
		return nil, &PatternError{code: "unsupported_node"}
	}
}
func Evaluate(input map[string]any) (any, error) {
	root, err := buildVisitor(input["tree"].(map[string]any))
	if err != nil {
		return nil, err
	}
	switch input["operation"] {
	case "evaluate":
		context, _ := input["context"].(map[string]any)
		return map[string]any{"result": root.Accept(EvalVisitor{context})}, nil
	case "collect_fields":
		return map[string]any{"fields": root.Accept(FieldVisitor{})}, nil
	default:
		return nil, &PatternError{code: "unsupported_operation"}
	}
}
