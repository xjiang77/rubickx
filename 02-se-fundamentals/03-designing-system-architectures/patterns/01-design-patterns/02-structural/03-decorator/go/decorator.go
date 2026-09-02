package decorator

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Client interface {
	Send(map[string]any) (map[string]any, error)
}
type baseClient struct{}

func cloneRequest(request map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range request {
		out[k] = v
	}
	headers := map[string]any{}
	if raw, ok := request["headers"].(map[string]any); ok {
		for k, v := range raw {
			headers[k] = v
		}
	}
	out["headers"] = headers
	applied := []string{}
	if raw, ok := request["applied"].([]string); ok {
		applied = append(applied, raw...)
	}
	out["applied"] = applied
	return out
}
func (baseClient) Send(request map[string]any) (map[string]any, error) {
	value := cloneRequest(request)
	return map[string]any{"path": value["path"], "headers": value["headers"], "applied": value["applied"]}, nil
}

type authDecorator struct {
	next  Client
	token string
}

func (d authDecorator) Send(request map[string]any) (map[string]any, error) {
	if d.token == "" {
		return nil, &PatternError{code: "missing_token"}
	}
	value := cloneRequest(request)
	value["headers"].(map[string]any)["Authorization"] = "Bearer " + d.token
	value["applied"] = append(value["applied"].([]string), "auth")
	return d.next.Send(value)
}

type traceDecorator struct{ next Client }

func (d traceDecorator) Send(request map[string]any) (map[string]any, error) {
	trace, _ := request["trace_id"].(string)
	if trace == "" {
		return nil, &PatternError{code: "missing_trace"}
	}
	value := cloneRequest(request)
	value["headers"].(map[string]any)["X-Trace-Id"] = trace
	value["applied"] = append(value["applied"].([]string), "trace")
	return d.next.Send(value)
}
func Evaluate(input map[string]any) (any, error) {
	var client Client = baseClient{}
	rawDecorators, _ := input["decorators"].([]any)
	for i := len(rawDecorators) - 1; i >= 0; i-- {
		switch rawDecorators[i].(string) {
		case "auth":
			token, _ := input["token"].(string)
			client = authDecorator{client, token}
		case "trace":
			client = traceDecorator{client}
		default:
			return nil, &PatternError{code: "unsupported_decorator"}
		}
	}
	rawRequests, _ := input["requests"].([]any)
	responses := make([]map[string]any, 0, len(rawRequests))
	for _, raw := range rawRequests {
		response, err := client.Send(raw.(map[string]any))
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return map[string]any{"responses": responses}, nil
}
