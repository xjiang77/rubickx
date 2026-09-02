package state

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type State interface {
	Name() string
	Handle(string) (State, error)
}
type state struct {
	name        string
	transitions map[string]string
}

func (s state) Name() string { return s.name }
func (s state) Handle(event string) (State, error) {
	target, ok := s.transitions[event]
	if !ok {
		return nil, &PatternError{code: "invalid_transition"}
	}
	return states()[target], nil
}
func states() map[string]State {
	return map[string]State{"queued": state{"queued", map[string]string{"start": "running", "cancel": "cancelled"}}, "running": state{"running", map[string]string{"complete": "completed", "fail": "failed"}}, "failed": state{"failed", map[string]string{"retry": "running", "cancel": "cancelled"}}, "completed": state{"completed", map[string]string{}}, "cancelled": state{"cancelled", map[string]string{}}}
}
func Evaluate(input map[string]any) (any, error) {
	current, ok := states()[input["initial"].(string)]
	if !ok {
		return nil, &PatternError{code: "unknown_state"}
	}
	history := []string{current.Name()}
	raw, _ := input["events"].([]any)
	for _, event := range raw {
		next, err := current.Handle(event.(string))
		if err != nil {
			return nil, err
		}
		current = next
		history = append(history, current.Name())
	}
	return map[string]any{"final": current.Name(), "history": history}, nil
}
