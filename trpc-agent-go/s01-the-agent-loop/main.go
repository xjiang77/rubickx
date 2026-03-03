// s01 - The Agent Loop (trpc-agent-go)
//
// Same concept as go/s01, but the framework handles the loop for you:
//
//	Runner → LLMAgent → [Tool Dispatch] → Event Stream
//
//	+----------+      +--------+      +---------+      +--------+
//	|   User   | ---> | Runner | ---> |LLMAgent | ---> |  Tool  |
//	|  prompt  |      |        |      |  (loop) |      |execute |
//	+----------+      +--------+      +----+----+      +---+----+
//	                      ^                ^               |
//	                      |  event stream  |  tool_result  |
//	                      +----------------+---------------+
//
// Key insight: "The loop is built into LLMAgent. You declare tools and stream events."
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
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

var guided = []struct {
	step   string
	prompt string
}{
	{"Watch the agent call bash", `Create a file called hello.go with a main function that prints "Hello, World!"`},
	{"See it build and run Go code", `Build and run hello.go`},
	{"Multi-step task with tool calls", `Create a directory called calc/ with a main.go that adds two numbers from command-line args, then test it with "go run ./calc/ 3 5"`},
	{"Explore project structure", `List all Go source files in the current directory tree`},
}

// ---------- Tool input type ----------

// BashInput is the input schema for the bash tool.
// The struct fields are auto-converted to JSON Schema by FunctionTool.
type BashInput struct {
	Command string `json:"command" description:"Shell command to execute"`
}

// ---------- Tool implementation ----------

func runBash(_ context.Context, input BashInput) (string, error) {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(input.Command, d) {
			return "Error: Dangerous command blocked", nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", input.Command)
	cmd.Dir, _ = os.Getwd()
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

// ---------- Event consumer ----------

// consumeEvents reads all events from the runner and prints text output.
// Returns the final assistant text response.
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
			// Print streaming delta content
			if choice.Delta.Content != "" {
				fmt.Print(choice.Delta.Content)
			}
			// Capture final message content
			if choice.Message.Content != "" {
				lastText = choice.Message.Content
			}
			// Show tool calls being made
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Function.Name != "" {
					fmt.Printf("\033[33m$ [tool: %s]\033[0m\n", tc.Function.Name)
				}
			}
		}
	}
	return lastText
}

func main() {
	// -- Model setup --
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

	// -- Tool setup --
	cwd, _ := os.Getwd()

	bashTool := function.NewFunctionTool(runBash,
		function.WithName("bash"),
		function.WithDescription("Run a shell command."),
	)

	// -- Agent setup --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)),
		llmagent.WithTools([]tool.Tool{bashTool}),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(20),
	)

	// -- Runner setup --
	sess := inmemory.NewSessionService()
	r := runner.NewRunner("rubickx", ag, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s01: The Agent Loop (trpc-agent-go)")
	fmt.Print("  \"The loop is built-in. You declare tools and stream events.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms01 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms01 >> \033[0m")
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
