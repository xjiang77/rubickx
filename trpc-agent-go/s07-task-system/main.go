// s07 - Task System (trpc-agent-go)
//
// Tasks persist as JSON files in .tasks/ so they survive context compression.
// Each task has a dependency graph (BlockedBy/Blocks).
// FunctionTools wrap CRUD operations; completing a task auto-unblocks downstream tasks.
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

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// ========== TaskManager ==========

// Task represents a single task persisted as a JSON file.
type Task struct {
	ID          int    `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"`    // "pending", "in_progress", "completed"
	BlockedBy   []int  `json:"blockedBy"` // IDs of tasks that must complete first
	Blocks      []int  `json:"blocks"`    // IDs of tasks this task is blocking
	Owner       string `json:"owner"`
}

// TaskManager provides CRUD + dependency graph over a .tasks/ directory.
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

func (tm *TaskManager) save(task *Task) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", task.ID))
	return os.WriteFile(path, data, 0o644)
}

// Create makes a new task with given subject and description.
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
		// Bidirectional: update the blocked tasks' BlockedBy lists
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

// clearDependency removes completedID from all other tasks' BlockedBy lists.
// Must be called with tm.mu held.
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
	workdir string
	tasks   *TaskManager
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Create dependent tasks", `Create 3 tasks: "Initialize Go module", "Implement core logic", "Write tests". Make them depend on each other in order.`},
	{"Inspect the graph", `List all tasks and show the dependency graph`},
	{"See unblocking in action", `Complete task 1 and then list tasks to see task 2 unblocked`},
	{"Parallel task graph", `Create a task board for a Go project: "parse config" -> "build server" + "build client" (parallel) -> "integration test"`},
}

// ========== Input types for task FunctionTools ==========

type TaskCreateInput struct {
	Subject     string `json:"subject" description:"Short task title"`
	Description string `json:"description,omitempty" description:"Detailed task description"`
}

type TaskUpdateInput struct {
	TaskID       int    `json:"task_id" description:"ID of the task to update"`
	Status       string `json:"status,omitempty" description:"New status: pending, in_progress, or completed"`
	AddBlockedBy []int  `json:"addBlockedBy,omitempty" description:"Task IDs that must complete before this task"`
	AddBlocks    []int  `json:"addBlocks,omitempty" description:"Task IDs that this task blocks"`
}

type TaskGetInput struct {
	TaskID int `json:"task_id" description:"ID of the task to retrieve"`
}

type TaskListInput struct{}

// ========== Base tool input types ==========

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

func handleTaskCreate(_ context.Context, input TaskCreateInput) (string, error) {
	result := tasks.Create(input.Subject, input.Description)
	fmt.Printf("\n\033[33m[task created]\033[0m\n")
	return result, nil
}

func handleTaskUpdate(_ context.Context, input TaskUpdateInput) (string, error) {
	result := tasks.Update(input.TaskID, input.Status, input.AddBlockedBy, input.AddBlocks)
	if input.Status == "completed" {
		fmt.Printf("\n\033[33m[task #%d completed, dependencies cleared]\033[0m\n", input.TaskID)
	}
	return result, nil
}

func handleTaskGet(_ context.Context, input TaskGetInput) (string, error) {
	return tasks.Get(input.TaskID), nil
}

func handleTaskList(_ context.Context, _ TaskListInput) (string, error) {
	result := tasks.ListAll()
	fmt.Printf("\n\033[33m--- Tasks ---\n%s\n---\033[0m\n", result)
	return result, nil
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

	// Initialize task manager backed by .tasks/ directory
	tasks = NewTaskManager(filepath.Join(workdir, ".tasks"))

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

	// -- Tools: base tools + task tools --
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
		function.NewFunctionTool(handleTaskCreate,
			function.WithName("task_create"),
			function.WithDescription("Create a new task with subject and optional description."),
		),
		function.NewFunctionTool(handleTaskUpdate,
			function.WithName("task_update"),
			function.WithDescription("Update a task's status or dependencies. Completing a task auto-unblocks downstream tasks."),
		),
		function.NewFunctionTool(handleTaskList,
			function.WithName("task_list"),
			function.WithDescription("List all tasks with their status and dependency information."),
		),
		function.NewFunctionTool(handleTaskGet,
			function.WithName("task_get"),
			function.WithDescription("Get full details of a specific task by ID."),
		),
	}

	// -- Agent --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			`You are a coding agent at %s.
Use task_create to plan work. Use task_update to track progress.
Completing a task automatically unblocks its dependents.
Tasks persist in .tasks/ and survive context resets.`, workdir)),
		llmagent.WithTools(tools),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(30),
	)

	// -- Runner --
	sess := inmemory.NewSessionService()
	r := runner.NewRunner("rubickx", ag, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s07: Task System (trpc-agent-go)")
	fmt.Print("  \"State that survives compression -- because it's outside the conversation.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  -> %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms07 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  Guided tour complete. Free mode -- type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms07 >> \033[0m")
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
