// s04 - Subagents
//
// Spawn a child agent with fresh messages=[]. The child works in its own
// context, sharing the filesystem, then returns only a summary to the parent.
//
//	Parent agent                     Subagent
//	+------------------+             +------------------+
//	| messages=[...]   |             | messages=[]      |  <-- fresh
//	|                  |  dispatch   |                  |
//	| tool: task       | ----------> | while tool_use:  |
//	|   prompt="..."   |             |   call tools     |
//	|   description="" |             |   append results |
//	|                  |  summary    |                  |
//	|   result = "..." | <---------- | return last text |
//	+------------------+             +------------------+
//	          |
//	Parent context stays clean.
//	Subagent context is discarded.
//
// Key insight: "Process isolation gives context isolation for free."
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== Globals ==========

var (
	model          string
	workdir        string
	parentSystem   string
	subagentSystem string
	childTools     []anthropic.ToolUnionParam
	parentTools    []anthropic.ToolUnionParam
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Delegate a research task", `Use a subtask to read go.mod and summarize what each dependency does`},
	{"Delegate file analysis", `Delegate: read all .go files in this directory tree and summarize what each one does`},
	{"Subagent creates, parent verifies", `Use a subtask to create a Go HTTP handler in server/handler.go, then verify it compiles from here`},
}

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()
	parentSystem = fmt.Sprintf("You are a coding agent at %s. Use the task tool to delegate exploration or subtasks.", workdir)
	subagentSystem = fmt.Sprintf("You are a coding subagent at %s. Complete the given task, then summarize your findings.", workdir)

	// -- Child tools: base tools only (no recursive spawning) --
	childTools = []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "bash",
			Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"command": map[string]interface{}{"type": "string"},
				},
				Required: []string{"command"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "read_file",
			Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":  map[string]interface{}{"type": "string"},
					"limit": map[string]interface{}{"type": "integer"},
				},
				Required: []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "write_file",
			Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":    map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "edit_file",
			Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":     map[string]interface{}{"type": "string"},
					"old_text": map[string]interface{}{"type": "string"},
					"new_text": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "old_text", "new_text"},
			},
		}},
	}

	// -- Parent tools: child tools + task dispatcher --
	parentTools = append(childTools, anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "task",
			Description: anthropic.String("Spawn a subagent with fresh context. It shares the filesystem but not conversation history."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"prompt":      map[string]interface{}{"type": "string"},
					"description": map[string]interface{}{"type": "string", "description": "Short description of the task"},
				},
				Required: []string{"prompt"},
			},
		},
	})

	// -- Dispatch map: base tools only (task handled separately) --
	dispatch = map[string]toolHandler{
		"bash":       handleBash,
		"read_file":  handleReadFile,
		"write_file": handleWriteFile,
		"edit_file":  handleEditFile,
	}
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

// ========== Tool handlers ==========

func handleBash(raw json.RawMessage) string {
	var input struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &input)
	return runBash(input.Command)
}

func handleReadFile(raw json.RawMessage) string {
	var input struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}
	_ = json.Unmarshal(raw, &input)
	return runRead(input.Path, input.Limit)
}

func handleWriteFile(raw json.RawMessage) string {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal(raw, &input)
	return runWrite(input.Path, input.Content)
}

func handleEditFile(raw json.RawMessage) string {
	var input struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	_ = json.Unmarshal(raw, &input)
	return runEdit(input.Path, input.OldText, input.NewText)
}

// ========== Tool implementations ==========

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
	cmd.Dir = workdir
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

func runRead(path string, limit int) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	if limit > 0 && limit < len(lines) {
		lines = append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit))
	}
	out := strings.Join(lines, "\n")
	if len(out) > 50000 {
		return out[:50000]
	}
	return out
}

func runWrite(path string, content string) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Wrote %d bytes", len(content))
}

func runEdit(path string, oldText string, newText string) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, oldText) {
		return fmt.Sprintf("Error: Text not found in %s", path)
	}
	newContent := strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(fp, []byte(newContent), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Edited %s", path)
}

// ========== Subagent ==========

// runSubagent spawns a child agent with fresh context.
// It runs its own agent loop, then returns only the final text summary.
// The child's message history is discarded after completion.
func runSubagent(client *anthropic.Client, prompt string) string {
	// Fresh context — no parent history
	subMessages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	for i := 0; i < 30; i++ { // safety limit
		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			System:    []anthropic.TextBlockParam{{Text: subagentSystem}},
			Messages:  subMessages,
			Tools:     childTools, // no "task" tool — prevents recursive spawning
			MaxTokens: 8000,
		})
		if err != nil {
			return fmt.Sprintf("Subagent error: %v", err)
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
		subMessages = append(subMessages, anthropic.NewAssistantMessage(assistantBlocks...))

		if resp.StopReason != "tool_use" {
			// Extract final text and return — child context discarded
			var texts []string
			for _, block := range resp.Content {
				if block.Type == "text" && block.Text != "" {
					texts = append(texts, block.Text)
				}
			}
			if len(texts) == 0 {
				return "(no summary)"
			}
			return strings.Join(texts, "")
		}

		// Execute tools
		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				handler, ok := dispatch[block.Name]
				var output string
				if ok {
					output = handler(block.Input)
				} else {
					output = fmt.Sprintf("Unknown tool: %s", block.Name)
				}
				if len(output) > 50000 {
					output = output[:50000]
				}
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		subMessages = append(subMessages, anthropic.NewUserMessage(results...))
	}

	return "(subagent hit iteration limit)"
}

// ========== Parent Agent Loop ==========

func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			System:    []anthropic.TextBlockParam{{Text: parentSystem}},
			Messages:  messages,
			Tools:     parentTools,
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

		if resp.StopReason != "tool_use" {
			return messages
		}

		// Execute tools — task gets special handling
		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				var output string
				if block.Name == "task" {
					var input struct {
						Prompt      string `json:"prompt"`
						Description string `json:"description"`
					}
					_ = json.Unmarshal(block.Input, &input)
					desc := input.Description
					if desc == "" {
						desc = "subtask"
					}
					fmt.Printf("> task (%s): %s\n", desc, truncate(input.Prompt, 80))
					output = runSubagent(client, input.Prompt)
				} else {
					handler, ok := dispatch[block.Name]
					if ok {
						output = handler(block.Input)
					} else {
						output = fmt.Sprintf("Unknown tool: %s", block.Name)
					}
				}
				fmt.Printf("  %s\n", truncate(output, 200))
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ========== Main ==========

func main() {
	opts := []option.RequestOption{}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)

	fmt.Println("\n  s04: Subagents")
	fmt.Print("  \"Process isolation gives context isolation for free.\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms04 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms04 >> \033[0m")
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
