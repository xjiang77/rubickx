package command

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type RouteTable struct{ routes map[string]string }
type Command interface {
	Execute()
	Undo()
}
type setRoute struct {
	receiver             *RouteTable
	key, value, previous string
	existed              bool
}

func (c *setRoute) Execute() {
	c.previous, c.existed = c.receiver.routes[c.key]
	c.receiver.routes[c.key] = c.value
}
func (c *setRoute) Undo() {
	if c.existed {
		c.receiver.routes[c.key] = c.previous
	} else {
		delete(c.receiver.routes, c.key)
	}
}

type Invoker struct{ history []Command }

func (i *Invoker) Run(c Command) { c.Execute(); i.history = append(i.history, c) }
func (i *Invoker) Undo() error {
	if len(i.history) == 0 {
		return &PatternError{code: "no_history"}
	}
	last := i.history[len(i.history)-1]
	i.history = i.history[:len(i.history)-1]
	last.Undo()
	return nil
}
func Evaluate(input map[string]any) (any, error) {
	routes := map[string]string{}
	if initial, ok := input["initial"].(map[string]any); ok {
		for k, v := range initial {
			routes[k] = v.(string)
		}
	}
	table := &RouteTable{routes}
	invoker := &Invoker{}
	raw, _ := input["commands"].([]any)
	for _, item := range raw {
		v := item.(map[string]any)
		invoker.Run(&setRoute{receiver: table, key: v["key"].(string), value: v["value"].(string)})
	}
	undo := int(input["undo"].(float64))
	for n := 0; n < undo; n++ {
		if err := invoker.Undo(); err != nil {
			return nil, err
		}
	}
	out := map[string]any{}
	for k, v := range table.routes {
		out[k] = v
	}
	return map[string]any{"routes": out, "history": len(invoker.history)}, nil
}
