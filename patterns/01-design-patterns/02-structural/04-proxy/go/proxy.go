package proxy

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Subject interface{ Read(string) (string, error) }
type documentStore struct{ documents map[string]string }

func newStore() *documentStore {
	return &documentStore{documents: map[string]string{"public": "status:ok", "secret": "key:rotated"}}
}
func (s *documentStore) Read(name string) (string, error) {
	value, ok := s.documents[name]
	if !ok {
		return "", &PatternError{code: "not_found"}
	}
	return value, nil
}

type documentProxy struct {
	role      string
	subject   Subject
	loadCount int
}

func (p *documentProxy) Read(name string) (string, error) {
	if name == "secret" && p.role != "admin" {
		return "", &PatternError{code: "forbidden"}
	}
	if p.subject == nil {
		p.subject = newStore()
		p.loadCount++
	}
	return p.subject.Read(name)
}
func Evaluate(input map[string]any) (any, error) {
	role, _ := input["role"].(string)
	p := &documentProxy{role: role}
	raw, _ := input["reads"].([]any)
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		value, err := p.Read(item.(string))
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return map[string]any{"values": values, "load_count": p.loadCount}, nil
}
