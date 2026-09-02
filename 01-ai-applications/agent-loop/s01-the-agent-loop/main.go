// s01 - The Agent Loop
//
// The entire secret of an AI coding agent in one pattern:
//
//	for stopReason == "tool_use" {
//	    response = LLM(messages, tools)
//	    execute tools
//	    append results
//	}
//
//	+----------+      +-------+      +---------+
//	|   User   | ---> |  LLM  | ---> |  Tool   |
//	|  prompt  |      |       |      | execute |
//	+----------+      +---+---+      +----+----+
//	                      ^               |
//	                      |   tool_result |
//	                      +---------------+
//	                      (loop continues)
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

var (
	model  string
	system string
	tools  []anthropic.ToolUnionParam
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

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	cwd, _ := os.Getwd()
	system = fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)

	tools = []anthropic.ToolUnionParam{
		anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "bash",
				Description: anthropic.String("Run a shell command."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"command": map[string]interface{}{
							"type": "string",
						},
					},
					Required: []string{"command"},
				},
			},
		},
	}
}

func runBash(command string) string {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir, _ = os.Getwd()
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (120s)"
	}

	out := strings.TrimSpace(string(output))
	if err != nil && out == "" {
		return fmt.Sprintf("Error: %v", err)
	}
	if out == "" {
		return "(no output)"
	}
	if len(out) > 50000 {
		return out[:50000]
	}
	return out
}

type bashInput struct {
	Command string `json:"command"`
}

// agentLoop is the core pattern: call the LLM with tools until it stops.
func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			System:    []anthropic.TextBlockParam{{Text: system}},
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 8000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "API error: %v\n", err)
			return messages
		}

		// Append assistant turn
		assistantBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(resp.Content))
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
			case "tool_use":
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))
			}
		}
		messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))

		// If the model didn't call a tool, we're done
		if resp.StopReason != "tool_use" {
			return messages
		}

		// Execute each tool call, collect results
		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				var input bashInput
				_ = json.Unmarshal(block.Input, &input)
				fmt.Printf("\033[33m$ %s\033[0m\n", input.Command)
				output := runBash(input.Command)
				if len(output) > 200 {
					fmt.Println(output[:200])
				} else {
					fmt.Println(output)
				}
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
}

func main() {
	opts := []option.RequestOption{}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)

	fmt.Println("\n  s01: The Agent Loop")
	fmt.Print("  \"One loop & Bash is all you need\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

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
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
			messages = agentLoop(&client, messages)
		} else {
			if input == "" {
				continue
			}
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(input)))
			messages = agentLoop(&client, messages)
		}

		// Print the last assistant response text
		if len(messages) > 0 {
			last := messages[len(messages)-1]
			for _, part := range last.Content {
				if part.OfText != nil {
					fmt.Println(part.OfText.Text)
				}
			}
		}
		fmt.Println()
	}
}
