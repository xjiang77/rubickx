// s08 - Background Tasks
//
// Run commands in background goroutines. A notification queue is drained
// before each LLM call to deliver results.
//
//	Main goroutine              Background goroutine
//	+-----------------+        +-----------------+
//	| agent loop      |        | task executes   |
//	| ...             |        | ...             |
//	| [LLM call] <---+------- | enqueue(result) |
//	|  ^drain queue   |        +-----------------+
//	+-----------------+
//
//	Timeline:
//	Agent ----[spawn A]----[spawn B]----[other work]----
//	               |              |
//	               v              v
//	            [A runs]      [B runs]        (parallel)
//	               |              |
//	               +-- notification queue --> [results injected]
//
// Key insight: "Fire and forget -- the agent doesn't block while the command runs."
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== BackgroundManager ==========

// bgTask tracks a single background command.
type bgTask struct {
	Status  string `json:"status"`  // running | completed | timeout | error
	Result  string `json:"result"`
	Command string `json:"command"`
}

// notification is pushed to the queue when a background task finishes.
type notification struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Command string `json:"command"`
	Result  string `json:"result"`
}

// BackgroundManager manages goroutine-based background execution with a
// thread-safe notification queue.
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

// Check returns the status of one task, or lists all if taskID is empty.
func (bg *BackgroundManager) Check(taskID string) string {
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

// DrainNotifications returns and clears all pending completion notifications.
func (bg *BackgroundManager) DrainNotifications() []notification {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	notifs := make([]notification, len(bg.queue))
	copy(notifs, bg.queue)
	bg.queue = bg.queue[:0]
	return notifs
}

// randomID generates an 8-char hex ID (like Python's uuid4()[:8]).
func randomID() string {
	const chars = "0123456789abcdef"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// ========== Globals ==========

var (
	model   string
	system  string
	workdir string
	bg      *BackgroundManager
	tools   []anthropic.ToolUnionParam
)

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()

	bg = NewBackgroundManager()

	system = fmt.Sprintf("You are a coding agent at %s. Use background_run for long-running commands.", workdir)

	// -- Tool definitions --
	tools = []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "bash",
			Description: anthropic.String("Run a shell command (blocking)."),
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
			Name:        "background_run",
			Description: anthropic.String("Run command in background goroutine. Returns task_id immediately."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"command": map[string]interface{}{"type": "string"},
				},
				Required: []string{"command"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "check_background",
			Description: anthropic.String("Check background task status. Omit task_id to list all."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string"},
				},
			},
		}},
	}

	// -- Dispatch map --
	dispatch = map[string]toolHandler{
		"bash":             handleBash,
		"read_file":        handleReadFile,
		"write_file":       handleWriteFile,
		"edit_file":        handleEditFile,
		"background_run":   handleBackgroundRun,
		"check_background": handleCheckBackground,
	}
}

// ========== Path safety ==========

func safePath(p string) (string, error) {
	full := filepath.Join(workdir, p)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", p)
	}
	if !strings.HasPrefix(abs, workdir) {
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

func handleBackgroundRun(raw json.RawMessage) string {
	var input struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &input)
	return bg.Run(input.Command)
}

func handleCheckBackground(raw json.RawMessage) string {
	var input struct {
		TaskID string `json:"task_id"`
	}
	_ = json.Unmarshal(raw, &input)
	return bg.Check(input.TaskID)
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

// ========== Agent Loop ==========

func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		// ---- Drain background notifications before each LLM call ----
		notifs := bg.DrainNotifications()
		if len(notifs) > 0 && len(messages) > 0 {
			var parts []string
			for _, n := range notifs {
				parts = append(parts, fmt.Sprintf("[bg:%s] %s: %s", n.TaskID, n.Status, n.Result))
			}
			notifText := strings.Join(parts, "\n")
			messages = append(messages,
				anthropic.NewUserMessage(
					anthropic.NewTextBlock(fmt.Sprintf("<background-results>\n%s\n</background-results>", notifText)),
				),
				anthropic.NewAssistantMessage(
					anthropic.NewTextBlock("Noted background results."),
				),
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

// ========== Main ==========

func main() {
	opts := []option.RequestOption{}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[36ms08 >> \033[0m")
		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())
		if query == "" || query == "q" || query == "exit" {
			break
		}

		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		messages = agentLoop(&client, messages)

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
