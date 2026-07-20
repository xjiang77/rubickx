package flyweight

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type ModelMetadata struct {
	Model, Provider string
	ContextWindow   int
}
type Factory struct{ values map[string]*ModelMetadata }

func NewFactory() *Factory { return &Factory{values: map[string]*ModelMetadata{}} }
func (f *Factory) Register(value map[string]any) error {
	candidate := &ModelMetadata{value["model"].(string), value["provider"].(string), int(value["context_window"].(float64))}
	if current, ok := f.values[candidate.Model]; ok && (current.Provider != candidate.Provider || current.ContextWindow != candidate.ContextWindow) {
		return &PatternError{code: "intrinsic_conflict"}
	}
	f.values[candidate.Model] = candidate
	return nil
}
func (f *Factory) Get(model string) (*ModelMetadata, error) {
	value, ok := f.values[model]
	if !ok {
		return nil, &PatternError{code: "unknown_model"}
	}
	return value, nil
}
func Evaluate(input map[string]any) (any, error) {
	factory := NewFactory()
	for _, raw := range input["definitions"].([]any) {
		if err := factory.Register(raw.(map[string]any)); err != nil {
			return nil, err
		}
	}
	rawRoutes, _ := input["routes"].([]any)
	routes := make([]map[string]any, 0, len(rawRoutes))
	values := make([]*ModelMetadata, 0, len(rawRoutes))
	for _, raw := range rawRoutes {
		route := raw.(map[string]any)
		metadata, err := factory.Get(route["model"].(string))
		if err != nil {
			return nil, err
		}
		values = append(values, metadata)
		routes = append(routes, map[string]any{"model": metadata.Model, "provider": metadata.Provider, "context_window": metadata.ContextWindow, "tenant": route["tenant"]})
	}
	reused := len(values) > 1
	if reused {
		for _, value := range values[1:] {
			if value != values[0] {
				reused = false
				break
			}
		}
	}
	return map[string]any{"routes": routes, "flyweight_count": len(factory.values), "reused": reused}, nil
}
