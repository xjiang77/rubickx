package builder

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type ChatRequest struct {
	Model     string
	Messages  []string
	MaxTokens int
	Stream    bool
}
type ChatRequestBuilder struct {
	model     string
	messages  []string
	maxTokens int
	stream    bool
}

func (b *ChatRequestBuilder) Configure(value map[string]any) *ChatRequestBuilder {
	b.model, _ = value["model"].(string)
	b.messages = nil
	for _, m := range value["messages"].([]any) {
		b.messages = append(b.messages, m.(string))
	}
	b.maxTokens = int(value["max_tokens"].(float64))
	b.stream, _ = value["stream"].(bool)
	return b
}
func (b *ChatRequestBuilder) Build() (ChatRequest, error) {
	if b.model == "" || len(b.messages) == 0 || b.maxTokens <= 0 {
		return ChatRequest{}, &PatternError{code: "invalid_request"}
	}
	result := ChatRequest{b.model, append([]string(nil), b.messages...), b.maxTokens, b.stream}
	*b = ChatRequestBuilder{}
	return result, nil
}
func Evaluate(input map[string]any) (any, error) {
	b := &ChatRequestBuilder{}
	requests := []map[string]any{}
	for _, raw := range input["builds"].([]any) {
		product, err := b.Configure(raw.(map[string]any)).Build()
		if err != nil {
			return nil, err
		}
		requests = append(requests, map[string]any{"model": product.Model, "messages": product.Messages, "max_tokens": product.MaxTokens, "stream": product.Stream})
	}
	return map[string]any{"requests": requests}, nil
}
