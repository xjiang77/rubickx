package observer

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Observer interface {
	Name() string
	Notify(string) error
}
type namedObserver struct{ name string }

func (o namedObserver) Name() string { return o.name }
func (o namedObserver) Notify(string) error {
	if o.name == "failing" {
		return &PatternError{code: "observer_failed"}
	}
	return nil
}

type Subject struct{ observers []Observer }

func (s *Subject) Publish(event string) []map[string]any {
	out := make([]map[string]any, 0, len(s.observers))
	for _, o := range append([]Observer(nil), s.observers...) {
		if err := o.Notify(event); err != nil {
			out = append(out, map[string]any{"event": event, "observer": o.Name(), "status": "failed", "error": err.(*PatternError).Code()})
		} else {
			out = append(out, map[string]any{"event": event, "observer": o.Name(), "status": "delivered"})
		}
	}
	return out
}
func (s *Subject) Unsubscribe(names map[string]bool) {
	kept := []Observer{}
	for _, o := range s.observers {
		if !names[o.Name()] {
			kept = append(kept, o)
		}
	}
	s.observers = kept
}
func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["observers"].([]any)
	subject := &Subject{}
	for _, name := range raw {
		value := name.(string)
		if value != "audit" && value != "metrics" && value != "failing" {
			return nil, &PatternError{code: "unsupported_observer"}
		}
		subject.observers = append(subject.observers, namedObserver{value})
	}
	receipts := []map[string]any{}
	events, _ := input["events"].([]any)
	for index, event := range events {
		receipts = append(receipts, subject.Publish(event.(string))...)
		if index == 0 {
			names := map[string]bool{}
			if values, ok := input["unsubscribe_after_first"].([]any); ok {
				for _, v := range values {
					names[v.(string)] = true
				}
			}
			subject.Unsubscribe(names)
		}
	}
	return map[string]any{"receipts": receipts, "active": len(subject.observers)}, nil
}
