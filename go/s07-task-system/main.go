// s07 - Task System
//
// Tasks persist as JSON files in .tasks/ so they survive context compression.
// Each task has a dependency graph (blockedBy/blocks).
//
//	.tasks/
//	  task_1.json  {"id":1, "subject":"...", "status":"completed", ...}
//	  task_2.json  {"id":2, "blockedBy":[1], "status":"pending", ...}
//	  task_3.json  {"id":3, "blockedBy":[2], "blocks":[], ...}
//
//	Dependency resolution:
//	+----------+     +----------+     +----------+
//	| task 1   | --> | task 2   | --> | task 3   |
//	| complete |     | blocked  |     | blocked  |
//	+----------+     +----------+     +----------+
//	     |                ^
//	     +--- completing task 1 removes it from task 2's blockedBy
//
// Key insight: "State that survives compression -- because it's outside the conversation."
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== TaskManager ==========

// Task represents a single task persisted as a JSON file.
type Task struct {
	ID          int    `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"`
	BlockedBy   []int  `json:"blockedBy"`
	Blocks      []int  `json:"blocks"`
	Owner       string `json:"owner"`
}

// TaskManager provides CRUD + dependency graph over .tasks/ directory.
type TaskManager struct {
	dir    string
	nextID int
	mu     sync.Mutex
}

// NewTaskManager creates a TaskManager, ensuring the directory exists.
func NewTaskManager(dir string) *TaskManager {
	_ = os.MkdirAll(dir, 0o755)
	tm := &TaskManager{dir: dir}
	tm.nextID = tm.maxID() + 1
	return tm
}

// maxID scans existing task files to find the highest ID.
func (tm *TaskManager) maxID() int {
	entries, _ := os.ReadDir(tm.dir)
	maxVal := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "task_") && strings.HasSuffix(name, ".json") {
			numStr := strings.TrimSuffix(strings.TrimPrefix(name, "task_"), ".json")
			if n, err := strconv.Atoi(numStr); err == nil && n > maxVal {
				maxVal = n
			}
		}
	}
	return maxVal
}

// load reads a task JSON file by ID.
func (tm *TaskManager) load(taskID int) (*Task, error) {
	path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", taskID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("task %d not found", taskID)
	}
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("corrupt task file: %v", err)
	}
	return &task, nil
}

// save writes a task to its JSON file.
func (tm *TaskManager) save(task *Task) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", task.ID))
	return os.WriteFile(path, data, 0o644)
}

// Create makes a new task with the given subject and description.
func (tm *TaskManager) Create(subject, description string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task := &Task{
		ID:          tm.nextID,
		Subject:     subject,
		Description: description,
		Status:      "pending",
		BlockedBy:   []int{},
		Blocks:      []int{},
		Owner:       "",
	}
	if err := tm.save(task); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	tm.nextID++
	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

// Get returns the full JSON for a task by ID.
func (tm *TaskManager) Get(taskID int) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, err := tm.load(taskID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

// Update modifies a task's status and/or dependencies.
func (tm *TaskManager) Update(taskID int, status string, addBlockedBy, addBlocks []int) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, err := tm.load(taskID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if status != "" {
		switch status {
		case "pending", "in_progress", "completed":
			task.Status = status
			if status == "completed" {
				tm.clearDependency(taskID)
			}
		default:
			return fmt.Sprintf("Error: Invalid status '%s'", status)
		}
	}

	if len(addBlockedBy) > 0 {
		task.BlockedBy = uniqueAppend(task.BlockedBy, addBlockedBy)
	}

	if len(addBlocks) > 0 {
		task.Blocks = uniqueAppend(task.Blocks, addBlocks)
		// Bidirectional: also update the blocked tasks' blockedBy lists
		for _, blockedID := range addBlocks {
			blocked, err := tm.load(blockedID)
			if err != nil {
				continue
			}
			if !containsInt(blocked.BlockedBy, taskID) {
				blocked.BlockedBy = append(blocked.BlockedBy, taskID)
				_ = tm.save(blocked)
			}
		}
	}

	if err := tm.save(task); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

// clearDependency removes completedID from all other tasks' blockedBy lists.
func (tm *TaskManager) clearDependency(completedID int) {
	entries, _ := os.ReadDir(tm.dir)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "task_") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(tm.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		if containsInt(task.BlockedBy, completedID) {
			task.BlockedBy = removeInt(task.BlockedBy, completedID)
			_ = tm.save(&task)
		}
	}
}

// ListAll returns a summary of all tasks.
func (tm *TaskManager) ListAll() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	entries, _ := os.ReadDir(tm.dir)
	var tasks []Task
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "task_") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tm.dir, e.Name()))
		if err != nil {
			continue
		}
		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return "No tasks."
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })

	var lines []string
	for _, t := range tasks {
		marker := "[?]"
		switch t.Status {
		case "pending":
			marker = "[ ]"
		case "in_progress":
			marker = "[>]"
		case "completed":
			marker = "[x]"
		}
		line := fmt.Sprintf("%s #%d: %s", marker, t.ID, t.Subject)
		if len(t.BlockedBy) > 0 {
			line += fmt.Sprintf(" (blocked by: %v)", t.BlockedBy)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// ========== Helpers ==========

func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func removeInt(slice []int, val int) []int {
	result := make([]int, 0, len(slice))
	for _, v := range slice {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}

func uniqueAppend(existing, additions []int) []int {
	seen := map[int]bool{}
	for _, v := range existing {
		seen[v] = true
	}
	result := append([]int{}, existing...)
	for _, v := range additions {
		if !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}
	return result
}

// ========== Globals ==========

var (
	model   string
	system  string
	workdir string
	tasks   *TaskManager
	tools   []anthropic.ToolUnionParam
)

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

var guided = []struct {
	step   string
	prompt string
}{
	{"Create dependent tasks", `Create 3 tasks: "Initialize Go module", "Implement core logic", "Write tests". Make them depend on each other in order.`},
	{"Inspect the graph", `List all tasks and show the dependency graph`},
	{"See unblocking in action", `Complete task 1 and then list tasks to see task 2 unblocked`},
	{"Parallel task graph", `Create a task board for a Go project: "parse config" -> "build server" + "build client" (parallel) -> "integration test"`},
}

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()

	tasks = NewTaskManager(filepath.Join(workdir, ".tasks"))

	system = fmt.Sprintf("You are a coding agent at %s. Use task tools to plan and track work.", workdir)

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
			Name:        "task_create",
			Description: anthropic.String("Create a new task."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"subject":     map[string]interface{}{"type": "string"},
					"description": map[string]interface{}{"type": "string"},
				},
				Required: []string{"subject"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "task_update",
			Description: anthropic.String("Update a task's status or dependencies."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{"type": "integer"},
					"status": map[string]interface{}{
						"type": "string",
						"enum": []string{"pending", "in_progress", "completed"},
					},
					"addBlockedBy": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "integer"},
					},
					"addBlocks": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "integer"},
					},
				},
				Required: []string{"task_id"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "task_list",
			Description: anthropic.String("List all tasks with status summary."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "task_get",
			Description: anthropic.String("Get full details of a task by ID."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{"type": "integer"},
				},
				Required: []string{"task_id"},
			},
		}},
	}

	// -- Dispatch map --
	dispatch = map[string]toolHandler{
		"bash":        handleBash,
		"read_file":   handleReadFile,
		"write_file":  handleWriteFile,
		"edit_file":   handleEditFile,
		"task_create": handleTaskCreate,
		"task_update": handleTaskUpdate,
		"task_list":   handleTaskList,
		"task_get":    handleTaskGet,
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

func handleTaskCreate(raw json.RawMessage) string {
	var input struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(raw, &input)
	return tasks.Create(input.Subject, input.Description)
}

func handleTaskUpdate(raw json.RawMessage) string {
	var input struct {
		TaskID       int    `json:"task_id"`
		Status       string `json:"status"`
		AddBlockedBy []int  `json:"addBlockedBy"`
		AddBlocks    []int  `json:"addBlocks"`
	}
	_ = json.Unmarshal(raw, &input)
	return tasks.Update(input.TaskID, input.Status, input.AddBlockedBy, input.AddBlocks)
}

func handleTaskList(raw json.RawMessage) string {
	return tasks.ListAll()
}

func handleTaskGet(raw json.RawMessage) string {
	var input struct {
		TaskID int `json:"task_id"`
	}
	_ = json.Unmarshal(raw, &input)
	return tasks.Get(input.TaskID)
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

	fmt.Println("\n  s07: Task System")
	fmt.Print("  \"DAG-based dependency tracking\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms07 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Printf("\033[36ms07 >> \033[0m")
		}

		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())

		if query == "q" || query == "exit" {
			break
		}

		if guidedIdx < len(guided) {
			if query == "" {
				query = guided[guidedIdx].prompt
			}
			guidedIdx++
		} else {
			if query == "" {
				continue
			}
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
