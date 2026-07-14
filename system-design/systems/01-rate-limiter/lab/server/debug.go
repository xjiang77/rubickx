package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type DebugStackFrame struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type DebugVariable struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	Type               string `json:"type,omitempty"`
	VariablesReference int    `json:"variablesReference,omitempty"`
}

type DebugSource struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type DebugSnapshot struct {
	SessionID   string            `json:"sessionId"`
	Status      string            `json:"status"`
	Source      DebugSource       `json:"source"`
	Line        int               `json:"line"`
	StackFrames []DebugStackFrame `json:"stackFrames"`
	Locals      []DebugVariable   `json:"locals"`
}

type DebugSessionRequest struct {
	Algorithm        string             `json:"algorithm"`
	Config           map[string]float64 `json:"config"`
	RequestTimeline  []RequestPoint     `json:"requestTimeline"`
	BreakpointStepID string             `json:"breakpointStepId,omitempty"`
}

type DelveSessionManager struct {
	root     string
	mu       sync.Mutex
	sessions map[string]*delveSession
	starting int
	nextID   atomic.Uint64
}

type delveSession struct {
	id         string
	tempDir    string
	binary     string
	scenario   string
	source     string
	breakpoint int
	process    *exec.Cmd
	client     *dapClient
	threadID   int
	mu         sync.Mutex
	closed     bool
	timer      *time.Timer
}

func NewDelveSessionManager(root string) *DelveSessionManager {
	return &DelveSessionManager{root: root, sessions: map[string]*delveSession{}}
}

func (m *DelveSessionManager) Create(ctx context.Context, input DebugSessionRequest) (DebugSnapshot, error) {
	runRequest := RunRequest{Algorithm: input.Algorithm, Language: LanguageGo, Config: input.Config, RequestTimeline: input.RequestTimeline}
	if err := validateRunRequest(runRequest); err != nil {
		return DebugSnapshot{}, err
	}
	if len(input.RequestTimeline) == 0 {
		return DebugSnapshot{}, fmt.Errorf("debug requestTimeline must contain at least one item")
	}
	steps := debugSteps[input.Algorithm]
	if len(steps) == 0 {
		return DebugSnapshot{}, fmt.Errorf("algorithm %q is not debuggable", input.Algorithm)
	}
	if input.BreakpointStepID == "" {
		input.BreakpointStepID = steps[0]
	}
	if !containsString(steps, input.BreakpointStepID) {
		return DebugSnapshot{}, fmt.Errorf("breakpointStepId %q does not belong to %s", input.BreakpointStepID, input.Algorithm)
	}
	m.mu.Lock()
	if len(m.sessions)+m.starting >= 4 {
		m.mu.Unlock()
		return DebugSnapshot{}, fmt.Errorf("at most four debug sessions may run concurrently")
	}
	m.starting++
	m.mu.Unlock()
	operationContext, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	session, snapshot, err := m.start(operationContext, runRequest, input.BreakpointStepID)
	m.mu.Lock()
	m.starting--
	if err != nil {
		m.mu.Unlock()
		return DebugSnapshot{}, err
	}
	m.sessions[session.id] = session
	session.timer = time.AfterFunc(5*time.Minute, func() { m.expire(session.id) })
	m.mu.Unlock()
	return snapshot, nil
}

func (m *DelveSessionManager) start(ctx context.Context, runRequest RunRequest, breakpointStepID string) (*delveSession, DebugSnapshot, error) {
	if m.root == "" {
		root, err := LabRoot()
		if err != nil {
			return nil, DebugSnapshot{}, err
		}
		m.root = root
	}
	dlv, err := exec.LookPath("dlv")
	if err != nil {
		return nil, DebugSnapshot{}, fmt.Errorf("dlv is required for Go debug mode: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "rate-limiter-lab-dlv-")
	if err != nil {
		return nil, DebugSnapshot{}, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	binary := filepath.Join(tempDir, "debug-fixture")
	scenario := filepath.Join(tempDir, "scenario.json")
	payload, err := json.Marshal(runRequest)
	if err != nil {
		cleanup()
		return nil, DebugSnapshot{}, err
	}
	if err := os.WriteFile(scenario, payload, 0o600); err != nil {
		cleanup()
		return nil, DebugSnapshot{}, err
	}
	moduleRoot := filepath.Clean(filepath.Join(m.root, "..", "..", ".."))
	build := exec.CommandContext(ctx, "go", "build", "-gcflags=all=-N -l", "-o", binary, "./systems/01-rate-limiter/lab/server/debugfixture")
	build.Dir = moduleRoot
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		cleanup()
		return nil, DebugSnapshot{}, fmt.Errorf("build debug fixture: %w: %s", buildErr, strings.TrimSpace(string(output)))
	}
	source := filepath.Join(m.root, "server", "algorithms.go")
	breakpoint, err := findStepLine(source, breakpointStepID)
	if err != nil {
		cleanup()
		return nil, DebugSnapshot{}, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cleanup()
		return nil, DebugSnapshot{}, err
	}
	defer listener.Close()
	process := exec.Command(dlv, "dap", "--client-addr="+listener.Addr().String(), "--log=false")
	process.Dir = moduleRoot
	var stderr bytes.Buffer
	process.Stderr = &stderr
	process.Stdout = io.Discard
	if err := process.Start(); err != nil {
		cleanup()
		return nil, DebugSnapshot{}, err
	}
	conn, err := acceptDAP(ctx, listener, process)
	if err != nil {
		_ = process.Process.Kill()
		_ = process.Wait()
		cleanup()
		return nil, DebugSnapshot{}, fmt.Errorf("connect to dlv dap: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	session := &delveSession{
		id: fmt.Sprintf("debug-%d", m.nextID.Add(1)), tempDir: tempDir, binary: binary,
		scenario: scenario, source: source, breakpoint: breakpoint, process: process, client: newDAPClient(conn),
	}
	failed := true
	defer func() {
		if failed {
			_ = session.stop(context.Background())
		}
	}()
	if _, err := session.client.request(ctx, "initialize", map[string]any{
		"adapterID": "go", "pathFormat": "path", "linesStartAt1": true, "columnsStartAt1": true,
		"supportsVariableType": true, "supportsVariablePaging": true, "locale": "en-us",
	}); err != nil {
		return nil, DebugSnapshot{}, err
	}
	if _, err := session.client.request(ctx, "launch", map[string]any{
		"request": "launch", "mode": "exec", "program": binary, "args": []string{scenario}, "stopOnEntry": false,
	}); err != nil {
		return nil, DebugSnapshot{}, err
	}
	if _, err := session.client.waitEvent(ctx, "initialized"); err != nil {
		return nil, DebugSnapshot{}, err
	}
	if _, err := session.client.request(ctx, "setBreakpoints", map[string]any{
		"source":      map[string]any{"name": filepath.Base(source), "path": source},
		"breakpoints": []map[string]any{{"line": breakpoint}}, "sourceModified": false,
	}); err != nil {
		return nil, DebugSnapshot{}, err
	}
	if _, err := session.client.request(ctx, "setExceptionBreakpoints", map[string]any{"filters": []string{}}); err != nil {
		return nil, DebugSnapshot{}, err
	}
	if _, err := session.client.request(ctx, "configurationDone", map[string]any{}); err != nil {
		return nil, DebugSnapshot{}, err
	}
	stopped, err := session.client.waitEvent(ctx, "stopped")
	if err != nil {
		return nil, DebugSnapshot{}, err
	}
	session.threadID = eventThreadID(stopped)
	snapshot, err := session.snapshot(ctx)
	if err != nil {
		return nil, DebugSnapshot{}, err
	}
	failed = false
	return session, snapshot, nil
}

func (m *DelveSessionManager) Command(ctx context.Context, sessionID, command string) (DebugSnapshot, error) {
	m.mu.Lock()
	session := m.sessions[sessionID]
	m.mu.Unlock()
	if session == nil {
		return DebugSnapshot{}, fmt.Errorf("debug session %q not found", sessionID)
	}
	if command == "stop" {
		if err := m.Delete(ctx, sessionID); err != nil {
			return DebugSnapshot{}, err
		}
		return DebugSnapshot{SessionID: sessionID, Status: "stopped"}, nil
	}
	operationContext, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	snapshot, err := session.command(operationContext, command)
	if err == nil && session.timer != nil {
		session.timer.Reset(5 * time.Minute)
	}
	return snapshot, err
}

func (m *DelveSessionManager) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	session := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	if session == nil {
		return fmt.Errorf("debug session %q not found", sessionID)
	}
	if session.timer != nil {
		session.timer.Stop()
	}
	return session.stop(ctx)
}

func (m *DelveSessionManager) expire(sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = m.Delete(ctx, sessionID)
}

func (m *DelveSessionManager) CloseAll() error {
	m.mu.Lock()
	sessions := make([]*delveSession, 0, len(m.sessions))
	for id, session := range m.sessions {
		sessions = append(sessions, session)
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	var result error
	for _, session := range sessions {
		if session.timer != nil {
			session.timer.Stop()
		}
		if err := session.stop(context.Background()); err != nil && result == nil {
			result = err
		}
	}
	return result
}

func (s *delveSession) command(ctx context.Context, command string) (DebugSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return DebugSnapshot{}, fmt.Errorf("debug session is closed")
	}
	switch command {
	case "next", "continue":
		if _, err := s.client.request(ctx, command, map[string]any{"threadId": s.threadID, "singleThread": false}); err != nil {
			return DebugSnapshot{}, err
		}
		stopped, err := s.client.waitEvent(ctx, "stopped")
		if err != nil {
			return DebugSnapshot{}, err
		}
		s.threadID = eventThreadID(stopped)
	case "restart":
		if _, err := s.client.request(ctx, "restart", map[string]any{"arguments": map[string]any{
			"request": "launch", "mode": "exec", "program": s.binary, "args": []string{s.scenario}, "stopOnEntry": false, "rebuild": false,
		}}); err != nil {
			return DebugSnapshot{}, err
		}
		if _, err := s.client.waitEvent(ctx, "initialized"); err != nil {
			return DebugSnapshot{}, err
		}
		if _, err := s.client.request(ctx, "configurationDone", map[string]any{}); err != nil {
			return DebugSnapshot{}, err
		}
		stopped, err := s.client.waitEvent(ctx, "stopped")
		if err != nil {
			return DebugSnapshot{}, err
		}
		s.threadID = eventThreadID(stopped)
	default:
		return DebugSnapshot{}, fmt.Errorf("unsupported debug command %q", command)
	}
	return s.snapshot(ctx)
}

func (s *delveSession) snapshot(ctx context.Context) (DebugSnapshot, error) {
	body, err := s.client.request(ctx, "stackTrace", map[string]any{"threadId": s.threadID, "startFrame": 0, "levels": 20})
	if err != nil {
		return DebugSnapshot{}, err
	}
	rawFrames, _ := body["stackFrames"].([]any)
	frames := make([]DebugStackFrame, 0, len(rawFrames))
	for _, raw := range rawFrames {
		frame, _ := raw.(map[string]any)
		source, _ := frame["source"].(map[string]any)
		frames = append(frames, DebugStackFrame{ID: intValue(frame["id"]), Name: stringValue(frame["name"]), File: labRelativePath(stringValue(source["path"])), Line: intValue(frame["line"])})
	}
	if len(frames) == 0 {
		return DebugSnapshot{}, fmt.Errorf("Delve returned no stack frames")
	}
	variables := []DebugVariable{}
	scopes, err := s.client.request(ctx, "scopes", map[string]any{"frameId": frames[0].ID})
	if err == nil {
		if rawScopes, ok := scopes["scopes"].([]any); ok {
			for _, rawScope := range rawScopes {
				scope, _ := rawScope.(map[string]any)
				if reference := intValue(scope["variablesReference"]); reference > 0 {
					variablesBody, variablesErr := s.client.request(ctx, "variables", map[string]any{"variablesReference": reference, "start": 0, "count": 100})
					if variablesErr != nil {
						continue
					}
					for _, rawVariable := range anySlice(variablesBody["variables"]) {
						variable, _ := rawVariable.(map[string]any)
						variables = append(variables, DebugVariable{Name: stringValue(variable["name"]), Value: stringValue(variable["value"]), Type: stringValue(variable["type"]), VariablesReference: intValue(variable["variablesReference"])})
					}
				}
			}
		}
	}
	content, _ := os.ReadFile(s.source)
	return DebugSnapshot{SessionID: s.id, Status: "stopped", Source: DebugSource{Path: labRelativePath(s.source), Content: string(content)}, Line: frames[0].Line, StackFrames: frames, Locals: variables}, nil
}

func (s *delveSession) stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.client != nil {
		disconnectContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = s.client.request(disconnectContext, "disconnect", map[string]any{"terminateDebuggee": true})
		cancel()
		_ = s.client.close()
	}
	if s.process != nil && s.process.Process != nil {
		_ = s.process.Process.Kill()
		_ = s.process.Wait()
	}
	return os.RemoveAll(s.tempDir)
}

func findStepLine(path, stepID string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	line := 0
	found := false
	for scanner.Scan() {
		line++
		if found && strings.TrimSpace(scanner.Text()) != "" {
			return line, nil
		}
		if strings.Contains(scanner.Text(), "@step:"+stepID) {
			found = true
		}
	}
	return 0, fmt.Errorf("step anchor %q not found in %s", stepID, path)
}

func acceptDAP(ctx context.Context, listener net.Listener, process *exec.Cmd) (net.Conn, error) {
	if tcpListener, ok := listener.(*net.TCPListener); ok {
		deadline := time.Now().Add(10 * time.Second)
		if contextDeadline, hasDeadline := ctx.Deadline(); hasDeadline && contextDeadline.Before(deadline) {
			deadline = contextDeadline
		}
		_ = tcpListener.SetDeadline(deadline)
	}
	conn, err := listener.Accept()
	if err != nil {
		if process.ProcessState != nil && process.ProcessState.Exited() {
			return nil, fmt.Errorf("dlv exited before connecting to DAP client")
		}
		return nil, err
	}
	return conn, nil
}

func eventThreadID(event map[string]any) int {
	body, _ := event["body"].(map[string]any)
	threadID := intValue(body["threadId"])
	if threadID == 0 {
		threadID = 1
	}
	return threadID
}

func anySlice(value any) []any {
	values, _ := value.([]any)
	return values
}

var debugSteps = map[string][]string{
	AlgorithmFixedWindow:        {"fixed.locate-window", "fixed.decision"},
	AlgorithmSlidingWindowLog:   {"sliding-log.evict", "sliding-log.decision"},
	AlgorithmSlidingWindowCount: {"sliding-counter.rotate", "sliding-counter.estimate", "sliding-counter.decision"},
	AlgorithmTokenBucket:        {"token.refill", "token.decision"},
	AlgorithmLeakyBucket:        {"leaky.drain", "leaky.decision"},
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
