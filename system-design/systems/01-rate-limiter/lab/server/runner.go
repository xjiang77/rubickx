package server

import (
	"context"
	"fmt"
)

type GoRunner struct{}

func (GoRunner) Run(_ context.Context, request RunRequest) (RunResponse, error) {
	emitter, decisions, err := executeGoScenario(request)
	if err != nil {
		return RunResponse{}, err
	}
	response := RunResponse{
		Language:  LanguageGo,
		Algorithm: request.Algorithm,
		Events:    emitter.events,
		Decisions: decisions,
	}
	if response.Events == nil {
		response.Events = []TraceEvent{}
	}
	if err := enrichSource(&response, LanguageGo); err != nil {
		return RunResponse{}, fmt.Errorf("load Go source: %w", err)
	}
	return response, nil
}

func executeGoScenario(request RunRequest) (*traceEmitter, []Decision, error) {
	if err := validateRunRequest(request); err != nil {
		return nil, nil, err
	}
	emitter := &traceEmitter{}
	limiter, err := newAlgorithm(request.Algorithm, request.Config, emitter)
	if err != nil {
		return nil, nil, err
	}
	decisions := make([]Decision, 0, len(request.RequestTimeline))
	for _, point := range request.RequestTimeline {
		decision, allowErr := limiter.Allow(point)
		if allowErr != nil {
			return nil, nil, allowErr
		}
		decisions = append(decisions, decision)
	}
	return emitter, decisions, nil
}

// RunDebugScenario executes the same validated Go algorithm path used by
// GoRunner. It exists only for the fixed Delve launcher; it never evaluates
// source code or accepts a program path from the request.
func RunDebugScenario(request RunRequest) error {
	_, _, err := executeGoScenario(request)
	return err
}

func validateRunRequest(request RunRequest) error {
	if request.RequestTimeline == nil {
		return fmt.Errorf("requestTimeline must be an array")
	}
	if len(request.RequestTimeline) > 100 {
		return fmt.Errorf("requestTimeline must contain at most 100 items")
	}
	switch request.Algorithm {
	case AlgorithmFixedWindow, AlgorithmSlidingWindowLog, AlgorithmSlidingWindowCount:
		if _, _, err := windowConfig(request.Config); err != nil {
			return err
		}
	case AlgorithmTokenBucket, AlgorithmLeakyBucket:
		if _, _, err := rateConfig(request.Config); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported algorithm %q", request.Algorithm)
	}
	var last int64
	for index, point := range request.RequestTimeline {
		if _, err := validatePoint(point, last, index > 0); err != nil {
			return fmt.Errorf("requestTimeline[%d]: %w", index, err)
		}
		last = point.AtMs
	}
	return nil
}

type RunnerRegistry struct {
	runners map[string]LanguageRunner
}

func NewRunnerRegistry(root string) *RunnerRegistry {
	return &RunnerRegistry{runners: map[string]LanguageRunner{
		LanguageGo:         GoRunner{},
		LanguagePython:     NewSubprocessRunner(LanguagePython, root),
		LanguageJava:       NewSubprocessRunner(LanguageJava, root),
		LanguageJavaScript: NewSubprocessRunner(LanguageJavaScript, root),
	}}
}

func (r *RunnerRegistry) Runner(language string) (LanguageRunner, error) {
	runner, ok := r.runners[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language %q", language)
	}
	return runner, nil
}

func (r *RunnerRegistry) Register(language string, runner LanguageRunner) {
	r.runners[language] = runner
}
