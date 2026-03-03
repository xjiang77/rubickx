// s03 - TodoWrite
//
// The model tracks its own progress via a TodoManager. A nag reminder
// forces it to keep updating when it forgets.
//
//	+----------+      +-------+      +---------+
//	|   User   | ---> |  LLM  | ---> | Tools   |
//	|  prompt  |      |       |      | + todo  |
//	+----------+      +---+---+      +----+----+
//	                      ^               |
//	                      |   tool_result |
//	                      +---------------+
//	                            |
//	                +-----------+-----------+
//	                | TodoManager state     |
//	                | [ ] task A            |
//	                | [>] task B <- doing   |
//	                | [x] task C            |
//	                +-----------------------+
//	                            |
//	                if rounds_since_todo >= 3:
//	                  inject <reminder>
//
// Key insight: "The agent can track its own progress -- and I can see it."
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

// ========== TodoManager ==========

// TodoItem represents one task the agent is tracking.
type TodoItem struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Status string `json:"status"` // "pending", "in_progress", "completed"
}

// TodoManager is structured state the LLM writes to.
type TodoManager struct {
	Items []TodoItem
}

// Update validates and replaces the entire todo list. Returns rendered text.
func (t *TodoManager) Update(items []TodoItem) (string, error) {
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
			// valid
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

// Render returns a human-readable view of the todo list.
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
	model   string
	system  string
	workdir string
	tools   []anthropic.ToolUnionParam
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

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()
	system = fmt.Sprintf(`You are a coding agent at %s.
Use the todo tool to plan multi-step tasks. Mark in_progress before starting, completed when done.
Prefer tools over prose.`, workdir)

	// -- Tool definitions --
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
		{OfTool: &anthropic.ToolParam{
			Name:        "todo",
			Description: anthropic.String("Update task list. Track progress on multi-step tasks."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"id":     map[string]interface{}{"type": "string"},
								"text":   map[string]interface{}{"type": "string"},
								"status": map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
							},
							"required": []string{"id", "text", "status"},
						},
					},
				},
				Required: []string{"items"},
			},
		}},
	}

	// -- Dispatch map --
	dispatch = map[string]toolHandler{
		"bash":       handleBash,
		"read_file":  handleReadFile,
		"write_file": handleWriteFile,
		"edit_file":  handleEditFile,
		"todo":       handleTodo,
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

func handleTodo(raw json.RawMessage) string {
	var input struct {
		Items []TodoItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	result, err := todo.Update(input.Items)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return result
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

// ========== Agent Loop (with nag reminder) ==========

func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
	roundsSinceTodo := 0

	for {
		// Nag reminder: if 3+ rounds without a todo update, inject reminder
		if roundsSinceTodo >= 3 && len(messages) > 0 {
			last := &messages[len(messages)-1]
			// Prepend a text reminder into the last user message
			last.Content = append(
				[]anthropic.ContentBlockParamUnion{anthropic.NewTextBlock("<reminder>Update your todos.</reminder>")},
				last.Content...,
			)
		}

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

		if resp.StopReason != "tool_use" {
			return messages
		}

		// Execute tools via dispatch
		var results []anthropic.ContentBlockParamUnion
		usedTodo := false
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
				if block.Name == "todo" {
					usedTodo = true
				}
			}
		}
		if usedTodo {
			roundsSinceTodo = 0
		} else {
			roundsSinceTodo++
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

	fmt.Println("\n  s03: TodoWrite")
	fmt.Print("  \"The agent can track its own progress -- and I can see it.\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

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
