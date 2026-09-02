package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SubprocessRunner struct {
	Language string
	Root     string
	Timeout  time.Duration
	javaMu   sync.Mutex
}

const (
	maxRunnerStdout = 2 << 20
	maxRunnerStderr = 64 << 10
)

func NewSubprocessRunner(language, root string) *SubprocessRunner {
	return &SubprocessRunner{Language: language, Root: root, Timeout: 5 * time.Second}
}

func (r *SubprocessRunner) Run(ctx context.Context, request RunRequest) (RunResponse, error) {
	if err := validateRunRequest(request); err != nil {
		return RunResponse{}, err
	}
	if r.Root == "" {
		root, err := LabRoot()
		if err != nil {
			return RunResponse{}, err
		}
		r.Root = root
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return RunResponse{}, err
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command, err := r.command(commandContext)
	if err != nil {
		return RunResponse{}, err
	}
	command.Dir = r.Root
	command.Stdin = bytes.NewReader(append(payload, '\n'))
	stdout := &cappedBuffer{max: maxRunnerStdout}
	stderr := &cappedBuffer{max: maxRunnerStderr}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if commandContext.Err() != nil {
			return RunResponse{}, fmt.Errorf("%s runner timed out: %w", r.Language, commandContext.Err())
		}
		return RunResponse{}, fmt.Errorf("%s runner failed: %w: %s", r.Language, err, strings.TrimSpace(stderr.String()))
	}
	if stdout.truncated {
		return RunResponse{}, fmt.Errorf("%s runner stdout exceeded %d bytes", r.Language, maxRunnerStdout)
	}
	var envelope struct {
		Events    []TraceEvent `json:"events"`
		Decisions []Decision   `json:"decisions"`
		Error     *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	if err := decoder.Decode(&envelope); err != nil {
		return RunResponse{}, fmt.Errorf("decode %s runner response: %w; stdout=%q stderr=%q", r.Language, err, stdout.String(), stderr.String())
	}
	if envelope.Error != nil {
		return RunResponse{}, fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if envelope.Events == nil || envelope.Decisions == nil {
		return RunResponse{}, fmt.Errorf("%s runner returned an incomplete response", r.Language)
	}
	response := RunResponse{Language: r.Language, Algorithm: request.Algorithm, Events: envelope.Events, Decisions: envelope.Decisions}
	if err := enrichSource(&response, r.Language); err != nil {
		return RunResponse{}, fmt.Errorf("load %s source: %w", r.Language, err)
	}
	return response, nil
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	max       int
	truncated bool
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	remaining := b.max - b.buffer.Len()
	if remaining > 0 {
		toWrite := len(data)
		if toWrite > remaining {
			toWrite = remaining
		}
		_, _ = b.buffer.Write(data[:toWrite])
	}
	if len(data) > remaining {
		b.truncated = true
	}
	return len(data), nil
}

func (b *cappedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *cappedBuffer) String() string {
	value := b.buffer.String()
	if b.truncated {
		return value + "…(truncated)"
	}
	return value
}

func (r *SubprocessRunner) command(ctx context.Context) (*exec.Cmd, error) {
	switch r.Language {
	case LanguagePython:
		return exec.CommandContext(ctx, "python3", filepath.Join(r.Root, "runners", "python", "runner.py")), nil
	case LanguageJavaScript:
		return exec.CommandContext(ctx, "node", filepath.Join(r.Root, "runners", "js", "runner.mjs")), nil
	case LanguageJava:
		if err := r.compileJava(ctx); err != nil {
			return nil, err
		}
		return exec.CommandContext(ctx, "java", "-cp", filepath.Join(r.Root, ".build", "java"), "RateLimiterRunner"), nil
	default:
		return nil, fmt.Errorf("unsupported subprocess language %q", r.Language)
	}
}

func (r *SubprocessRunner) compileJava(ctx context.Context) error {
	r.javaMu.Lock()
	defer r.javaMu.Unlock()
	sources := []string{filepath.Join(r.Root, "runners", "java", "RateLimiterRunner.java")}
	if _, err := os.Stat(sources[0]); err != nil {
		return fmt.Errorf("no Java runner sources found")
	}
	outputDir := filepath.Join(r.Root, ".build", "java")
	classFile := filepath.Join(outputDir, "RateLimiterRunner.class")
	if javaBuildCurrent(classFile, sources) {
		return nil
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	args := append([]string{"-d", outputDir}, sources...)
	command := exec.CommandContext(ctx, "javac", args...)
	command.Dir = r.Root
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile Java runner: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func javaBuildCurrent(classFile string, sources []string) bool {
	classInfo, err := os.Stat(classFile)
	if err != nil {
		return false
	}
	for _, source := range sources {
		info, statErr := os.Stat(source)
		if statErr != nil || info.ModTime().After(classInfo.ModTime()) {
			return false
		}
	}
	return true
}
