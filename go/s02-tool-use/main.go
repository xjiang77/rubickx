// s02 - Tools
//
// The agent loop from s01 didn't change. We just added tools to the array
// and a dispatch map to route calls.
//
//	+----------+      +-------+      +------------------+
//	|   User   | ---> |  LLM  | ---> | Tool Dispatch    |
//	|  prompt  |      |       |      | {                |
//	+----------+      +---+---+      |   bash: runBash  |
//	                      ^          |   read: runRead  |
//	                      |          |   write: runWrite|
//	                      +----------+   edit: runEdit  |
//	                      tool_result| }                |
//	                                 +------------------+
//
// Key insight: "The loop didn't change at all. I just added tools."
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

var (
	model   string
	system  string
	workdir string
	tools   []anthropic.ToolUnionParam
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Use the read_file tool", `Read the file go.mod to see this project's dependencies`},
	{"Use write_file to create code", `Create a file called greet/greet.go with a package greet and a Hello(name string) string function`},
	{"Use edit_file for surgical changes", `Edit greet/greet.go to add a Goodbye function below the Hello function`},
	{"Verify with read_file", `Read greet/greet.go to verify both functions exist`},
}

// toolHandler is the signature for every tool executor.
type toolHandler func(input json.RawMessage) string

// dispatch maps tool names to their handlers.
var dispatch map[string]toolHandler

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()
	system = fmt.Sprintf("You are a coding agent at %s. Use tools to solve tasks. Act, don't explain.", workdir)

	// -- Tool definitions (JSON Schema) --
	tools = []anthropic.ToolUnionParam{
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

	// -- The dispatch map: {tool_name: handler} --
	dispatch = map[string]toolHandler{
		"bash":       handleBash,
		"read_file":  handleReadFile,
		"write_file": handleWriteFile,
		"edit_file":  handleEditFile,
	}
}

// ---------- Path safety ----------

// safePath resolves a path relative to workdir and ensures it doesn't escape.
func safePath(p string) (string, error) {
	full := filepath.Join(workdir, p)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", p)
	}
	// Ensure the resolved path is still under workdir
	if abs != workdir && !strings.HasPrefix(abs, workdir+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
}

// ---------- Tool handlers ----------

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

// ---------- Tool implementations ----------

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
	dir := filepath.Dir(fp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
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

// ---------- Agent Loop (unchanged from s01) ----------

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

		// Execute each tool call via dispatch map, collect results
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
				fmt.Printf("> %s: %s\n", block.Name, truncate(output, 200))
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

// ---------- Main ----------

func main() {
	opts := []option.RequestOption{}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)

	fmt.Println("\n  s02: Tool Use")
	fmt.Print("  \"The loop didn't change at all. I just added tools.\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms02 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms02 >> \033[0m")
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
