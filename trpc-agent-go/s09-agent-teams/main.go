// s09 - Agent Teams (trpc-agent-go)
//
// In go/s09, we manually managed goroutines and JSONL inboxes per teammate.
// With trpc-agent-go, we use llmagent.WithSubAgents() to register child agents.
// The lead agent delegates to sub-agents automatically via its tool set.
//
//	Lead Agent (orchestrator)
//	+-----------------------------+
//	| LLMAgent "lead"             |
//	| WithSubAgents([            ]|
//	|   researcher,               |
//	|   implementer               |
//	| ])                          |
//	+-------------+---------------+
//	              |  delegate via sub-agent tool call
//	     +--------+--------+
//	     |                 |
//	+----v-----+     +-----v----+
//	| researcher|     |implementer|
//	| (explore) |     | (coding)  |
//	+-----------+     +----------+
//
// Key insight: "WithSubAgents registers child agents as callable tools automatically."
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

	agentpkg "trpc.group/trpc-go/trpc-agent-go/agent"
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
	{"Delegate research to sub-agent", `Have the researcher sub-agent read go.mod and summarize the dependencies`},
	{"Delegate coding to implementer", `Have the implementer sub-agent create a simple Go file pkg/greet/greet.go with a Hello function`},
	{"Coordinated task", `Use both sub-agents: have researcher find all .go files, then have implementer add a comment header to one of them`},
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

	// -- Base tools (shared by all agents) --
	baseTools := []tool.Tool{
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

	// -- Sub-agent: researcher (exploration and reading) --
	// Uses IsolatedRequest so each delegation gets fresh context.
	researcher := llmagent.New("researcher",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are the researcher sub-agent at %s. "+
				"Explore the codebase, read files, and summarize findings. "+
				"Complete your task and return a clear summary.", workdir)),
		llmagent.WithTools(baseTools),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(20),
		llmagent.WithMessageFilterMode(llmagent.IsolatedRequest),
	)

	// -- Sub-agent: implementer (coding and writing) --
	implementer := llmagent.New("implementer",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are the implementer sub-agent at %s. "+
				"Write code, create files, and implement features. "+
				"Complete your task and return a summary of what you built.", workdir)),
		llmagent.WithTools(baseTools),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(20),
		llmagent.WithMessageFilterMode(llmagent.IsolatedRequest),
	)

	// -- Lead agent: orchestrates via WithSubAgents --
	// WithSubAgents registers each sub-agent as a callable tool automatically.
	// The lead can delegate by calling "researcher" or "implementer" as tools.
	leadAgent := llmagent.New("lead",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are the lead agent at %s. "+
				"Orchestrate tasks by delegating to your sub-agents: "+
				"'researcher' for exploration and reading, 'implementer' for coding and writing. "+
				"You can also use base tools directly. Coordinate the team to solve complex tasks.", workdir)),
		llmagent.WithTools(baseTools),
		llmagent.WithSubAgents([]agentpkg.Agent{researcher, implementer}),
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
	fmt.Println("\n  s09: Agent Teams (trpc-agent-go)")
	fmt.Print("  \"WithSubAgents registers child agents as callable tools automatically.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  --> %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms09 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  Guided tour complete. Free mode -- type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms09 >> \033[0m")
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
