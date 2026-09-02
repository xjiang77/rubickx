package abstractfactory

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type CloudFactory interface {
	Family() string
	Queue(string) string
	ObjectStore(string) string
}
type awsFactory struct{}

func (awsFactory) Family() string                   { return "aws" }
func (awsFactory) Queue(prefix string) string       { return "sqs:" + prefix }
func (awsFactory) ObjectStore(prefix string) string { return "s3:" + prefix }

type gcpFactory struct{}

func (gcpFactory) Family() string                   { return "gcp" }
func (gcpFactory) Queue(prefix string) string       { return "pubsub:" + prefix }
func (gcpFactory) ObjectStore(prefix string) string { return "gcs:" + prefix }

func Evaluate(input map[string]any) (any, error) {
	prefix := input["prefix"].(string)
	rawProviders, _ := input["providers"].([]any)
	resources := make([]map[string]any, 0, len(rawProviders))
	for _, raw := range rawProviders {
		var factory CloudFactory
		switch raw.(string) {
		case "aws":
			factory = awsFactory{}
		case "gcp":
			factory = gcpFactory{}
		default:
			return nil, &PatternError{code: "unsupported_provider"}
		}
		resources = append(resources, map[string]any{"family": factory.Family(), "queue": factory.Queue(prefix), "object_store": factory.ObjectStore(prefix)})
	}
	return map[string]any{"resources": resources}, nil
}
