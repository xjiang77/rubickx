// s03 - TodoWrite (trpc-agent-go)
//
// In go/s03, we manually injected nag reminders into the message array.
// With trpc-agent-go, we use Model Callbacks (BeforeModel) to intercept
// the LLM request and inject reminders automatically.
//
//	+----------+      +--------+      +---------+      +---------+
//	|   User   | ---> | Runner | ---> |LLMAgent | ---> | Tools   |
//	|  prompt  |      |        |      |  (loop) |      | + todo  |
//	+----------+      +--------+      +----+----+      +----+----+
//	                                       |                |
//	                            BeforeModel callback        |
//	                            injects <reminder>          |
//	                            if rounds >= 3              |
//	                                       |                |
//	                              +--------+--------+       |
//	                              | TodoManager     |<------+
//	                              | [ ] task A      |
//	                              | [>] task B      |
//	                              | [x] task C      |
//	                              +-----------------+
//
// Key insight: "Model callbacks inject behavior without touching the agent loop."
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	agentmodel "trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// ========== TodoManager ==========

type TodoItem struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Status string `json:"status"` // "pending", "in_progress", "completed"
}

type TodoManager struct {
	mu    sync.Mutex
	Items []TodoItem
}

func (t *TodoManager) Update(items []TodoItem) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(items) > 20 {
		return "", fmt.Errorf("max 20 todos allowed")
	}
	inProgressCount := 0
	for i := range items {
		item := &items[i]
		if item.Text == "" {
			return "", fmt.Errorf("item %s: text required", item.ID)
		}
		if item.ID == "" {
			item.ID = fmt.Sprintf("%d", i+1)
		}
		switch item.Status {
		case "pending", "in_progress", "completed":
		default:
			return "", fmt.Errorf("item %s: invalid status '%s'", item.ID, item.Status)
		}
		if item.Status == "in_progress" {
			inProgressCount++
		}
	}
	if inProgressCount > 1 {
		return "", fmt.Errorf("only one task can be in_progress at a time")
	}
	t.Items = items
	return t.Render(), nil
}

func (t *TodoManager) Render() string {
	if len(t.Items) == 0 {
		return "No todos."
	}
	markers := map[string]string{
		"pending":     "[ ]",
		"in_progress": "[>]",
		"completed":   "[x]",
	}
	var lines []string
	done := 0
	for _, item := range t.Items {
		lines = append(lines, fmt.Sprintf("%s #%s: %s", markers[item.Status], item.ID, item.Text))
		if item.Status == "completed" {
			done++
		}
	}
	lines = append(lines, fmt.Sprintf("\n(%d/%d completed)", done, len(t.Items)))
	return strings.Join(lines, "\n")
}

// ========== Globals ==========

var (
	workdir string
	todo    = &TodoManager{}
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Multi-step refactor with todo tracking", `Create a Go CLI tool in cmd/todo/ that manages a todo list stored in a JSON file, with add/list/done subcommands`},
	{"Project scaffolding task", `Create a Go package in pkg/mathutil/ with Add, Subtract, Multiply functions and a corresponding test file pkg/mathutil/mathutil_test.go`},
	{"Review task with multiple files", `Review all .go files we created and fix any issues — add missing error handling and improve naming`},
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
	OldText string `json:"old_text" description:"Exact text to find"`
	NewText string `json:"new_text" description:"Replacement text"`
}

type TodoInput struct {
	Items []TodoItem `json:"items" description:"Full replacement todo list"`
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

func handleTodo(_ context.Context, input TodoInput) (string, error) {
	result, err := todo.Update(input.Items)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	// Display the todo list in terminal
	fmt.Printf("\n\033[33m--- Todo List ---\n%s\n---\033[0m\n", result)
	return result, nil
}

// ========== Nag reminder tracker ==========

// nagTracker tracks rounds since last todo update for the nag reminder.
type nagTracker struct {
	mu              sync.Mutex
	roundsSinceTodo int
}

func (n *nagTracker) increment() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.roundsSinceTodo++
}

func (n *nagTracker) reset() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.roundsSinceTodo = 0
}

func (n *nagTracker) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.roundsSinceTodo
}

// ========== Event consumer ==========

func consumeEvents(events <-chan *event.Event, nag *nagTracker) string {
	var lastText string
	for ev := range events {
		if ev.IsRunnerCompletion() {
			break
		}
		if ev.Response == nil {
			continue
		}

		// Track tool usage for nag reminder
		if ev.Response.IsToolCallResponse() {
			for _, choice := range ev.Response.Choices {
				for _, tc := range choice.Message.ToolCalls {
					if tc.Function.Name == "todo" {
						nag.reset()
					}
				}
				for _, tc := range choice.Delta.ToolCalls {
					if tc.Function.Name == "todo" {
						nag.reset()
					}
				}
			}
		}
		if ev.Response.IsToolResultResponse() {
			nag.increment()
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

	// -- Nag reminder via Model Callbacks --
	nag := &nagTracker{}

	modelCBs := agentmodel.NewCallbacks()
	modelCBs.RegisterBeforeModel(
		func(ctx context.Context, args *agentmodel.BeforeModelArgs) (*agentmodel.BeforeModelResult, error) {
			if nag.count() >= 3 && args.Request != nil && len(args.Request.Messages) > 0 {
				// Inject reminder into the last message
				last := &args.Request.Messages[len(args.Request.Messages)-1]
				if last.Role == agentmodel.RoleUser || last.Role == agentmodel.RoleTool {
					last.Content = "<reminder>Update your todos before continuing.</reminder>\n" + last.Content
				}
			}
			return nil, nil
		},
	)

	// -- Tools --
	tools := []tool.Tool{
		function.NewFunctionTool(handleBash,
			function.WithName("bash"),
			function.WithDescription("Run a shell command."),
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
		function.NewFunctionTool(handleTodo,
			function.WithName("todo"),
			function.WithDescription("Update task list. Track progress on multi-step tasks. Send full replacement list."),
		),
	}

	// -- Agent (with model callbacks for nag reminder) --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			`You are a coding agent at %s.
Use the todo tool to plan multi-step tasks. Mark in_progress before starting, completed when done.
Prefer tools over prose.`, workdir)),
		llmagent.WithTools(tools),
		llmagent.WithModelCallbacks(modelCBs),
		llmagent.WithGenerationConfig(agentmodel.GenerationConfig{
			MaxTokens: agentmodel.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(30),
	)

	// -- Runner --
	sess := inmemory.NewSessionService()
	r := runner.NewRunner("rubickx", ag, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s03: TodoWrite (trpc-agent-go)")
	fmt.Print("  \"Model callbacks inject behavior without touching the agent loop.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms03 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms03 >> \033[0m")
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
			events, err := r.Run(context.Background(), "user", sessionID, agentmodel.NewUserMessage(query))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(events, nag)
			if text != "" {
				fmt.Println(text)
			}
		} else {
			if input == "" {
				continue
			}
			events, err := r.Run(context.Background(), "user", sessionID, agentmodel.NewUserMessage(input))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(events, nag)
			if text != "" {
				fmt.Println(text)
			}
		}
		fmt.Println()
	}
}
