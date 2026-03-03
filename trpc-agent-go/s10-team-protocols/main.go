// s10 - Team Protocols (trpc-agent-go)
//
// In go/s10, protocols were custom code (shutdown FSM, plan approval FSM).
// With trpc-agent-go, we use Agent/Tool/Model Callbacks for lifecycle hooks.
// Callbacks observe every event in the agent+tool execution without manual plumbing.
//
//	BeforeAgent          AfterAgent
//	    |                    |
//	+---v--------------------v---+
//	| LLMAgent "lead"            |
//	|                            |
//	|  BeforeTool   AfterTool    |
//	|      |            |        |
//	|  +---v----+   +---v----+   |
//	|  | tool   |   | result |   |
//	|  | exec   |   | logged |   |
//	|  +--------+   +--------+   |
//	+----------------------------+
//
//	Stats tracked per agent call:
//	  - total agent calls
//	  - total tool uses
//	  - tool use counts per tool name
//	  - approval check for "dangerous" tools
//
// Key insight: "Callbacks observe the whole execution graph without touching agent logic."
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
	"sync/atomic"
	"time"

	agentpkg "trpc.group/trpc-go/trpc-agent-go/agent"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	toolpkg "trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

var workdir string

var guided = []struct {
	step   string
	prompt string
}{
	{"Watch callbacks fire on every tool call", `List all .go files in this project and read go.mod`},
	{"Trigger the approval protocol", `Run the command: echo dangerous_operation`},
	{"Check stats", `/stats`},
}

// ========== Stats tracker ==========

// Stats tracks agent execution statistics, thread-safe via atomic/mutex.
type Stats struct {
	AgentCalls int64
	ToolUses   int64
	Errors     int64

	mu         sync.Mutex
	ToolCounts map[string]int64
}

func NewStats() *Stats {
	return &Stats{
		ToolCounts: make(map[string]int64),
	}
}

func (s *Stats) RecordAgentCall() {
	atomic.AddInt64(&s.AgentCalls, 1)
}

func (s *Stats) RecordToolUse(toolName string) {
	atomic.AddInt64(&s.ToolUses, 1)
	s.mu.Lock()
	s.ToolCounts[toolName]++
	s.mu.Unlock()
}

func (s *Stats) RecordError() {
	atomic.AddInt64(&s.Errors, 1)
}

func (s *Stats) Report() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent calls: %d\n", atomic.LoadInt64(&s.AgentCalls)))
	sb.WriteString(fmt.Sprintf("Tool uses:   %d\n", atomic.LoadInt64(&s.ToolUses)))
	sb.WriteString(fmt.Sprintf("Errors:      %d\n", atomic.LoadInt64(&s.Errors)))
	if len(s.ToolCounts) > 0 {
		sb.WriteString("Tool breakdown:\n")
		for name, count := range s.ToolCounts {
			sb.WriteString(fmt.Sprintf("  %-20s %d\n", name, count))
		}
	}
	return sb.String()
}

// ========== Approval protocol ==========

// approvalRequired lists tool names that require approval before execution.
// In a real system this would be configurable. We use a simple flag here.
var approvalRequired = map[string]bool{
	"bash": false, // toggled via /approve command
}

// approvalMu protects approvalRequired
var approvalMu sync.Mutex

func setApprovalRequired(toolName string, required bool) {
	approvalMu.Lock()
	defer approvalMu.Unlock()
	approvalRequired[toolName] = required
}

func isApprovalRequired(toolName string) bool {
	approvalMu.Lock()
	defer approvalMu.Unlock()
	return approvalRequired[toolName]
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

	// -- Stats tracker (shared across callbacks) --
	stats := NewStats()

	// -- Agent Callbacks: observe agent lifecycle --
	agentCBs := agentpkg.NewCallbacks()
	agentCBs.RegisterBeforeAgent(func(ctx context.Context, args *agentpkg.BeforeAgentArgs) (*agentpkg.BeforeAgentResult, error) {
		stats.RecordAgentCall()
		fmt.Printf("\033[35m[agent:before] %s invoked\033[0m\n", args.Invocation.Agent.Info().Name)
		return nil, nil
	})
	agentCBs.RegisterAfterAgent(func(ctx context.Context, args *agentpkg.AfterAgentArgs) (*agentpkg.AfterAgentResult, error) {
		if args.Error != nil {
			stats.RecordError()
			fmt.Printf("\033[31m[agent:after] %s error: %v\033[0m\n", args.Invocation.Agent.Info().Name, args.Error)
		}
		return nil, nil
	})

	// -- Tool Callbacks: observe tool execution + approval protocol --
	toolCBs := toolpkg.NewCallbacks()
	toolCBs.RegisterBeforeTool(func(ctx context.Context, args *toolpkg.BeforeToolArgs) (*toolpkg.BeforeToolResult, error) {
		// Approval protocol: if approval is required for this tool, block it
		if isApprovalRequired(args.ToolName) {
			fmt.Printf("\033[31m[tool:blocked] '%s' requires approval. Use /approve to enable.\033[0m\n", args.ToolName)
			return &toolpkg.BeforeToolResult{
				CustomResult: fmt.Sprintf("Tool '%s' blocked: approval required. Inform the user.", args.ToolName),
			}, nil
		}
		fmt.Printf("\033[33m[tool:before] %s\033[0m\n", args.ToolName)
		return nil, nil
	})
	toolCBs.RegisterAfterTool(func(ctx context.Context, args *toolpkg.AfterToolArgs) (*toolpkg.AfterToolResult, error) {
		stats.RecordToolUse(args.ToolName)
		if args.Error != nil {
			stats.RecordError()
		}
		return nil, nil
	})

	// -- Base tools --
	baseTools := []toolpkg.Tool{
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
	}

	// -- Sub-agents (researcher + implementer, same as s09) --
	researcher := llmagent.New("researcher",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are the researcher sub-agent at %s. Explore, read, and summarize.", workdir)),
		llmagent.WithTools(baseTools),
		llmagent.WithToolCallbacks(toolCBs),
		llmagent.WithAgentCallbacks(agentCBs),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(15),
		llmagent.WithMessageFilterMode(llmagent.IsolatedRequest),
	)

	implementer := llmagent.New("implementer",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are the implementer sub-agent at %s. Write code and create files.", workdir)),
		llmagent.WithTools(baseTools),
		llmagent.WithToolCallbacks(toolCBs),
		llmagent.WithAgentCallbacks(agentCBs),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(15),
		llmagent.WithMessageFilterMode(llmagent.IsolatedRequest),
	)

	// -- Lead agent with callbacks + sub-agents --
	leadAgent := llmagent.New("lead",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are the lead agent at %s. "+
				"Delegate to 'researcher' for exploration and 'implementer' for coding. "+
				"You can also use tools directly.", workdir)),
		llmagent.WithTools(baseTools),
		llmagent.WithSubAgents([]agentpkg.Agent{researcher, implementer}),
		llmagent.WithAgentCallbacks(agentCBs),
		llmagent.WithToolCallbacks(toolCBs),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(20),
	)

	// -- Runner --
	sess := inmemory.NewSessionService()
	r := runner.NewRunner("rubickx", leadAgent, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s10: Team Protocols (trpc-agent-go)")
	fmt.Print("  \"Callbacks observe the whole execution graph without touching agent logic.\"\n\n")
	fmt.Println("  Commands: /stats, /approve (toggle bash approval), /deny")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  --> %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms10 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  Guided tour complete. Free mode -- type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms10 >> \033[0m")
		}

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "q" || input == "exit" {
			break
		}

		// Local commands
		if input == "/stats" {
			fmt.Printf("\n\033[36m[stats]\033[0m\n%s\n", stats.Report())
			continue
		}
		if input == "/approve" {
			setApprovalRequired("bash", true)
			fmt.Println("  [protocol] bash tool now requires approval.")
			continue
		}
		if input == "/deny" {
			setApprovalRequired("bash", false)
			fmt.Println("  [protocol] bash tool approval requirement removed.")
			continue
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
