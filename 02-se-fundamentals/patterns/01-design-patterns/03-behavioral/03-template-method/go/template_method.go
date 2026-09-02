package templatemethod

import "strings"

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Transformer interface {
	Format() string
	Transform(string) []string
}
type csvJob struct{}

func (csvJob) Format() string              { return "csv" }
func (csvJob) Transform(v string) []string { return strings.Split(v, ",") }

type jsonJob struct{}

func (jsonJob) Format() string              { return "json" }
func (jsonJob) Transform(v string) []string { return strings.Split(v, "|") }
func run(job Transformer, payload string) (map[string]any, error) {
	if payload == "" {
		return nil, &PatternError{code: "invalid_payload"}
	}
	return map[string]any{"format": job.Format(), "data": job.Transform(payload), "steps": []string{"validate", "transform:" + job.Format(), "persist"}}, nil
}
func Evaluate(input map[string]any) (any, error) {
	raw, _ := input["jobs"].([]any)
	results := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		value := item.(map[string]any)
		var job Transformer
		switch value["format"] {
		case "csv":
			job = csvJob{}
		case "json":
			job = jsonJob{}
		default:
			return nil, &PatternError{code: "unsupported_format"}
		}
		result, err := run(job, value["payload"].(string))
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return map[string]any{"results": results}, nil
}
