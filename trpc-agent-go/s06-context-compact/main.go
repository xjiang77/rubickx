// s06 - Context Compact (trpc-agent-go)
//
// The framework provides session-level memory management.
// WithSessionEventLimit caps stored events; WithAddSessionSummary auto-injects
// a summary when the session is full. WithMaxHistoryRuns limits the history window.
// A compact FunctionTool lets the model force a fresh session at any time.
//
//	Session service (inmemory):
//	+----------------------------------+
//	| EventLimit=100                   |
//	| When full: summarize + trim      |  <-- auto-compact via WithAddSessionSummary
//	+----------------------------------+
//	         |
//	         v
//	+----------------------------------+
//	| LLMAgent                         |
//	| WithMaxHistoryRuns(10)           |  <-- sliding window on message history
//	| WithAddSessionSummary(true)      |  <-- inject summary into prompt
//	+----------------------------------+
//	         |
//	         v
//	+----------------------------------+
//	| compact tool (FunctionTool)      |
//	| -> start new session             |  <-- manual compact triggered by model
//	+----------------------------------+
//
// Key insight: "The framework handles memory. You configure limits."
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

var workdir string

var guided = []struct {
	step   string
	prompt string
}{
	{"Build up conversation context", `Read every .go file in this project tree and summarize what each one does`},
	{"Continue working in the bounded window", `Now write a quick summary of all the patterns you observed across the files`},
	{"Trigger manual compact", `Use the compact tool to compress the conversation history`},
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

type CompactInput struct {
	Focus string `json:"focus,omitempty" description:"What to preserve in the new session summary"`
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

// ========== Session state (for compact tool) ==========

var (
	currentSessionID string
	sessionCounter   int
)

func newSessionID() string {
	sessionCounter++
	return fmt.Sprintf("session-%d", sessionCounter)
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

// handleCompact starts a new session, which triggers the framework's
// session summarization on the next run if WithAddSessionSummary is set.
func handleCompact(_ context.Context, input CompactInput) (string, error) {
	oldSession := currentSessionID
	currentSessionID = newSessionID()
	focus := input.Focus
	if focus == "" {
		focus = "all work done so far"
	}
	fmt.Printf("\n\033[33m[compact: %s -> %s | focus: %s]\033[0m\n", oldSession, currentSessionID, focus)
	return fmt.Sprintf("Session compacted. New session: %s. Focus preserved: %s.", currentSessionID, focus), nil
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

	// Initialize session tracking
	sessionCounter = 0
	currentSessionID = newSessionID()

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
		function.NewFunctionTool(handleCompact,
			function.WithName("compact"),
			function.WithDescription("Compress conversation history into a new session. Call when context is full."),
		),
	}

	// -- Agent with context management options --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			`You are a coding agent at %s. Use tools to solve tasks.
When your context feels full, call compact to start fresh with a summary.`, workdir)),
		llmagent.WithTools(tools),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(30),
		llmagent.WithAddSessionSummary(true), // auto-inject session summary
		llmagent.WithMaxHistoryRuns(10),      // sliding window: last 10 runs
	)

	// -- Runner with event-limited session for auto-summarization --
	sess := inmemory.NewSessionService(
		inmemory.WithSessionEventLimit(100), // trigger summary after 100 events
	)
	r := runner.NewRunner("rubickx", ag, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s06: Context Compact (trpc-agent-go)")
	fmt.Print("  \"The framework handles memory. You configure limits.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  -> %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms06 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  Guided tour complete. Free mode -- type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms06 >> \033[0m")
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
			events, err := r.Run(context.Background(), "user", currentSessionID, model.NewUserMessage(query))
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
			events, err := r.Run(context.Background(), "user", currentSessionID, model.NewUserMessage(input))
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
