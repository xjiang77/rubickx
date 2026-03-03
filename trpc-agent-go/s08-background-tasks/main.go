// s08 - Background Tasks (trpc-agent-go)
//
// In go/s08, we managed goroutines and a notification queue manually.
// With trpc-agent-go, we use WithLongRunning(true) on a FunctionTool to
// signal that bg_run may return before the work is done, and drain
// notifications at the top of each REPL loop.
//
//	Main goroutine              Background goroutine
//	+-----------------+        +-----------------+
//	| REPL loop       |        | task executes   |
//	|  drain notifs   |        | ...             |
//	|  r.Run(...)  <--+------- | enqueue(result) |
//	+-----------------+        +-----------------+
//
//	Timeline:
//	Agent ----[bg_run A]----[bg_run B]----[other work]----
//	                |              |
//	                v              v
//	             [A runs]      [B runs]      (parallel goroutines)
//	                |              |
//	                +----- notification queue --> [results injected]
//
// Key insight: "Fire and forget -- the agent doesn't block while the command runs."
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// ========== BackgroundManager ==========

// bgTask tracks a single background command.
type bgTask struct {
	Status  string `json:"status"` // running | completed | timeout | error
	Result  string `json:"result"`
	Command string `json:"command"`
}

// notification is pushed to the queue when a background task finishes.
type notification struct {
	TaskID  string
	Status  string
	Command string
	Result  string
}

// BackgroundManager manages goroutine-based background execution.
type BackgroundManager struct {
	tasks map[string]*bgTask
	queue []notification
	mu    sync.Mutex
}

// NewBackgroundManager creates an empty manager.
func NewBackgroundManager() *BackgroundManager {
	return &BackgroundManager{
		tasks: make(map[string]*bgTask),
	}
}

// Run starts a background goroutine and returns a task ID immediately.
func (bg *BackgroundManager) Run(command string) string {
	taskID := randomID()
	bg.mu.Lock()
	bg.tasks[taskID] = &bgTask{
		Status:  "running",
		Command: command,
	}
	bg.mu.Unlock()

	go bg.execute(taskID, command)

	cmdPreview := command
	if len(cmdPreview) > 80 {
		cmdPreview = cmdPreview[:80]
	}
	return fmt.Sprintf("Background task %s started: %s", taskID, cmdPreview)
}

// execute is the goroutine target: runs subprocess, captures output, pushes to queue.
func (bg *BackgroundManager) execute(taskID, command string) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()

	var status, result string
	if ctx.Err() == context.DeadlineExceeded {
		status = "timeout"
		result = "Error: Timeout (300s)"
	} else if err != nil {
		out := strings.TrimSpace(string(output))
		if out != "" {
			status = "completed"
			result = out
		} else {
			status = "error"
			result = fmt.Sprintf("Error: %v", err)
		}
	} else {
		status = "completed"
		result = strings.TrimSpace(string(output))
	}

	if result == "" {
		result = "(no output)"
	}
	if len(result) > 50000 {
		result = result[:50000]
	}

	bg.mu.Lock()
	defer bg.mu.Unlock()

	bg.tasks[taskID].Status = status
	bg.tasks[taskID].Result = result

	// Truncate result for notification (keep full in tasks map)
	notifResult := result
	if len(notifResult) > 500 {
		notifResult = notifResult[:500]
	}
	cmdPreview := command
	if len(cmdPreview) > 80 {
		cmdPreview = cmdPreview[:80]
	}

	bg.queue = append(bg.queue, notification{
		TaskID:  taskID,
		Status:  status,
		Command: cmdPreview,
		Result:  notifResult,
	})
}

// Status returns the status of one task, or lists all if taskID is empty.
func (bg *BackgroundManager) Status(taskID string) string {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	if taskID != "" {
		t, ok := bg.tasks[taskID]
		if !ok {
			return fmt.Sprintf("Error: Unknown task %s", taskID)
		}
		r := t.Result
		if r == "" {
			r = "(running)"
		}
		cmdPreview := t.Command
		if len(cmdPreview) > 60 {
			cmdPreview = cmdPreview[:60]
		}
		return fmt.Sprintf("[%s] %s\n%s", t.Status, cmdPreview, r)
	}

	var lines []string
	for tid, t := range bg.tasks {
		cmdPreview := t.Command
		if len(cmdPreview) > 60 {
			cmdPreview = cmdPreview[:60]
		}
		lines = append(lines, fmt.Sprintf("%s: [%s] %s", tid, t.Status, cmdPreview))
	}
	if len(lines) == 0 {
		return "No background tasks."
	}
	return strings.Join(lines, "\n")
}

// Result returns the full result of a completed task.
func (bg *BackgroundManager) Result(taskID string) string {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	t, ok := bg.tasks[taskID]
	if !ok {
		return fmt.Sprintf("Error: Unknown task %s", taskID)
	}
	if t.Status == "running" {
		return fmt.Sprintf("Task %s is still running.", taskID)
	}
	return fmt.Sprintf("[%s] %s", t.Status, t.Result)
}

// DrainNotifications returns and clears all pending completion notifications.
func (bg *BackgroundManager) DrainNotifications() []notification {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	notifs := make([]notification, len(bg.queue))
	copy(notifs, bg.queue)
	bg.queue = bg.queue[:0]
	return notifs
}

func randomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ========== Globals ==========

var (
	workdir string
	bg      *BackgroundManager
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Background exec + foreground work", `Run "sleep 3 && echo done" in the background, then create a hello.go file while it runs`},
	{"Multiple background tasks", `Start 2 background tasks: "sleep 2 && echo task1" and "sleep 4 && echo task2". Check their status.`},
	{"Real-world background usage", `Run "go vet ./..." in the background while you create a simple math.go file`},
}

// ========== Input types ==========

type BashInput struct {
	Command string `json:"command" description:"Shell command to execute"`
}

type ReadFileInput struct {
	Path  string `json:"path" description:"File path to read"`
	Limit int    `json:"limit,omitempty" description:"Max lines to return"`
}

type WriteFileInput struct {
	Path    string `json:"path" description:"File path to write"`
	Content string `json:"content" description:"Content to write"`
}

type EditFileInput struct {
	Path    string `json:"path" description:"File path to edit"`
	OldText string `json:"old_text" description:"Exact text to find and replace"`
	NewText string `json:"new_text" description:"Text to replace with"`
}

type BgRunInput struct {
	Command string `json:"command" description:"Shell command to run in background"`
}

type BgStatusInput struct {
	TaskID string `json:"task_id,omitempty" description:"Task ID to check, or empty to list all"`
}

type BgResultInput struct {
	TaskID string `json:"task_id" description:"Task ID to get full result for"`
}

// ========== Path safety ==========

func safePath(p string) (string, error) {
	full := filepath.Join(workdir, p)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", p)
	}
	if abs != workdir && !strings.HasPrefix(abs, workdir+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
}

// ========== Tool implementations ==========

func handleBash(_ context.Context, input BashInput) (string, error) {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(input.Command, d) {
			return "Error: Dangerous command blocked", nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", input.Command)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (120s)", nil
	}
	out := strings.TrimSpace(string(output))
	if err != nil && out == "" {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if out == "" {
		return "(no output)", nil
	}
	if len(out) > 50000 {
		return out[:50000], nil
	}
	return out, nil
}

func handleReadFile(_ context.Context, input ReadFileInput) (string, error) {
	fp, err := safePath(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	lines := strings.Split(string(data), "\n")
	if input.Limit > 0 && input.Limit < len(lines) {
		lines = append(lines[:input.Limit], fmt.Sprintf("... (%d more lines)", len(lines)-input.Limit))
	}
	out := strings.Join(lines, "\n")
	if len(out) > 50000 {
		return out[:50000], nil
	}
	return out, nil
}

func handleWriteFile(_ context.Context, input WriteFileInput) (string, error) {
	fp, err := safePath(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if err := os.WriteFile(fp, []byte(input.Content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(input.Content), input.Path), nil
}

func handleEditFile(_ context.Context, input EditFileInput) (string, error) {
	fp, err := safePath(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	content := string(data)
	if !strings.Contains(content, input.OldText) {
		return fmt.Sprintf("Error: Text not found in %s", input.Path), nil
	}
	newContent := strings.Replace(content, input.OldText, input.NewText, 1)
	if err := os.WriteFile(fp, []byte(newContent), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Edited %s", input.Path), nil
}

// handleBgRun starts a background command. WithLongRunning(true) signals to the
// framework that this tool may return before its work is done.
func handleBgRun(_ context.Context, input BgRunInput) (string, error) {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot"}
	for _, d := range dangerous {
		if strings.Contains(input.Command, d) {
			return "Error: Dangerous command blocked", nil
		}
	}
	return bg.Run(input.Command), nil
}

func handleBgStatus(_ context.Context, input BgStatusInput) (string, error) {
	return bg.Status(input.TaskID), nil
}

func handleBgResult(_ context.Context, input BgResultInput) (string, error) {
	return bg.Result(input.TaskID), nil
}

// ========== Event consumer ==========

func consumeEvents(events <-chan *event.Event) string {
	var lastText string
	for ev := range events {
		if ev.IsRunnerCompletion() {
			break
		}
		if ev.Response == nil {
			continue
		}
		for _, choice := range ev.Response.Choices {
			if choice.Delta.Content != "" {
				fmt.Print(choice.Delta.Content)
			}
			if choice.Message.Content != "" {
				lastText = choice.Message.Content
			}
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Function.Name != "" {
					fmt.Printf("\033[33m> %s\033[0m\n", tc.Function.Name)
				}
			}
		}
	}
	return lastText
}

// ========== Main ==========

func main() {
	workdir, _ = os.Getwd()
	bg = NewBackgroundManager()

	// -- Model --
	modelID := os.Getenv("MODEL_ID")
	if modelID == "" {
		modelID = "gpt-4o"
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	var modelOpts []openai.Option
	modelOpts = append(modelOpts, openai.WithAPIKey(apiKey))
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		modelOpts = append(modelOpts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelID, modelOpts...)

	// -- Tools --
	// bg_run uses WithLongRunning(true): the framework knows it returns before
	// the underlying work completes — the agent can proceed with other tools.
	tools := []tool.Tool{
		function.NewFunctionTool(handleBash,
			function.WithName("bash"),
			function.WithDescription("Run a shell command (blocking)."),
		),
		function.NewFunctionTool(handleReadFile,
			function.WithName("read_file"),
			function.WithDescription("Read file contents."),
		),
		function.NewFunctionTool(handleWriteFile,
			function.WithName("write_file"),
			function.WithDescription("Write content to file."),
		),
		function.NewFunctionTool(handleEditFile,
			function.WithName("edit_file"),
			function.WithDescription("Replace exact text in file."),
		),
		function.NewFunctionTool(handleBgRun,
			function.WithName("bg_run"),
			function.WithDescription("Run a command in the background. Returns task_id immediately without waiting."),
			function.WithLongRunning(true),
		),
		function.NewFunctionTool(handleBgStatus,
			function.WithName("bg_status"),
			function.WithDescription("Check background task status. Omit task_id to list all tasks."),
		),
		function.NewFunctionTool(handleBgResult,
			function.WithName("bg_result"),
			function.WithDescription("Get the full result of a completed background task by task_id."),
		),
	}

	// -- Agent --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are a coding agent at %s. "+
				"Use bg_run for long commands (go build, tests, linting). "+
				"Check bg_status to monitor tasks. Use bg_result to get full output. "+
				"Continue foreground work while background tasks run.", workdir)),
		llmagent.WithTools(tools),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(30),
	)

	// -- Runner --
	sess := inmemory.NewSessionService()
	r := runner.NewRunner("rubickx", ag, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s08: Background Tasks (trpc-agent-go)")
	fmt.Print("  \"Fire and forget -- the agent doesn't block while the command runs.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		// Drain background notifications before each prompt
		notifs := bg.DrainNotifications()
		if len(notifs) > 0 {
			fmt.Println("\n\033[32m[background notifications]\033[0m")
			for _, n := range notifs {
				fmt.Printf("  [%s] task %s (%s): %s\n", n.Status, n.TaskID, n.Command, truncate(n.Result, 120))
			}
			fmt.Println()
		}

		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms08 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  Guided tour complete. Free mode -- type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms08 >> \033[0m")
		}

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "q" || input == "exit" {
			break
		}

		if guidedIdx < len(guided) {
			query := input
			if query == "" {
				query = guided[guidedIdx].prompt
			}
			guidedIdx++
			events, err := r.Run(context.Background(), "user", sessionID, model.NewUserMessage(query))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(events)
			if text != "" {
				fmt.Println(text)
			}
		} else {
			if input == "" {
				continue
			}
			events, err := r.Run(context.Background(), "user", sessionID, model.NewUserMessage(input))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(events)
			if text != "" {
				fmt.Println(text)
			}
		}
		fmt.Println()
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
