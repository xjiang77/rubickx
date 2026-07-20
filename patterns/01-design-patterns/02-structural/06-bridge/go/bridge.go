package bridge

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Channel interface{ Send(string, string) string }
type emailChannel struct{}

func (emailChannel) Send(recipient, message string) string {
	return "email:" + recipient + ":" + message
}

type slackChannel struct{}

func (slackChannel) Send(recipient, message string) string {
	return "slack:" + recipient + ":" + message
}

type Alert interface{ Deliver(string, string) string }
type alert struct {
	prefix  string
	channel Channel
}

func (a alert) Deliver(recipient, body string) string {
	return a.channel.Send(recipient, a.prefix+" "+body)
}
func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["notifications"].([]any)
	deliveries := make([]string, 0, len(raw))
	for _, item := range raw {
		value := item.(map[string]any)
		var channel Channel
		switch value["channel"] {
		case "email":
			channel = emailChannel{}
		case "slack":
			channel = slackChannel{}
		default:
			return nil, &PatternError{code: "unsupported_channel"}
		}
		prefix := ""
		switch value["kind"] {
		case "incident":
			prefix = "[INCIDENT]"
		case "reminder":
			prefix = "[REMINDER]"
		default:
			return nil, &PatternError{code: "unsupported_alert"}
		}
		var current Alert = alert{prefix, channel}
		deliveries = append(deliveries, current.Deliver(value["recipient"].(string), value["body"].(string)))
	}
	return map[string]any{"deliveries": deliveries}, nil
}
