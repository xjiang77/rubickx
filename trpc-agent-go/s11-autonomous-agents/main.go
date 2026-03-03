// s11 - Autonomous Agents (trpc-agent-go)
//
// Work-stealing agents poll a shared task board and claim unclaimed tasks.
// The lead REPL creates tasks; a background worker agent picks them up.
//
//	Lead REPL                       Background Worker
//	+------------------+            +------------------+
//	| task_create tool | ---------> | .tasks/          |
//	| task_list tool   |            |  task_1.json     |
//	| task_get tool    |            |  task_2.json     |
//	+------------------+            +------------------+
//	                                        |
//	                                 poll every 5s
//	                                        |
//	                                        v
//	                                +------------------+
//	                                | worker LLMAgent  |
//	                                | claim → work     |
//	                                | → complete       |
//	                                +------------------+
//
// Key insight: "The agent finds work itself. The task board is the coordination point."
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

var workdir string

var guided = []struct {
	step   string
	prompt string
}{
	{"Create tasks for autonomous worker", `Create 3 tasks: "Write a hello.go file", "Add a greet function to hello.go", "Write tests for hello.go"`},
	{"Watch the worker claim tasks", `List tasks and tell me which ones the worker has already claimed`},
	{"Check task status", `List all tasks with their current status`},
}

// ========== Task types ==========

// Task represents a work item persisted as a JSON file.
type Task struct {
	ID          int    `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending, in_progress, completed
	Owner       string `json:"owner"`
	BlockedBy   []int  `json:"blockedBy"`
	Blocks      []int  `json:"blocks"`
}

// ========== TaskManager ==========

type TaskManager struct {
	dir    string
	nextID int
	mu     sync.Mutex
}

func NewTaskManager(dir string) *TaskManager {
	_ = os.MkdirAll(dir, 0o755)
	tm := &TaskManager{dir: dir}
	tm.nextID = tm.maxID() + 1
	return tm
}

func (tm *TaskManager) maxID() int {
	entries, _ := os.ReadDir(tm.dir)
	maxVal := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "task_") && strings.HasSuffix(name, ".json") {
			var n int
			if _, err := fmt.Sscanf(name, "task_%d.json", &n); err == nil && n > maxVal {
				maxVal = n
			}
		}
	}
	return maxVal
}

func (tm *TaskManager) taskPath(id int) string {
	return filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", id))
}

func (tm *TaskManager) load(id int) (*Task, error) {
	data, err := os.ReadFile(tm.taskPath(id))
	if err != nil {
		return nil, fmt.Errorf("task %d not found", id)
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("corrupt task %d", id)
	}
	return &t, nil
}

func (tm *TaskManager) save(t *Task) {
	data, _ := json.MarshalIndent(t, "", "  ")
	_ = os.WriteFile(tm.taskPath(t.ID), data, 0o644)
}

func (tm *TaskManager) Create(subject, description string, blockedBy []int) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	t := &Task{
		ID:          tm.nextID,
		Subject:     subject,
		Description: description,
		Status:      "pending",
		Owner:       "",
		BlockedBy:   blockedBy,
		Blocks:      []int{},
	}
	if t.BlockedBy == nil {
		t.BlockedBy = []int{}
	}
	tm.save(t)
	tm.nextID++
	out, _ := json.MarshalIndent(t, "", "  ")
	return string(out)
}

func (tm *TaskManager) Get(id int) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	t, err := tm.load(id)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(t, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) Update(id int, status, owner string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	t, err := tm.load(id)
	if err != nil {
		return "", err
	}
	if status != "" {
		valid := map[string]bool{"pending": true, "in_progress": true, "completed": true}
		if !valid[status] {
			return "", fmt.Errorf("invalid status: %s", status)
		}
		t.Status = status
	}
	if owner != "" {
		t.Owner = owner
	}
	tm.save(t)
	out, _ := json.MarshalIndent(t, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) Claim(id int, owner string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	t, err := tm.load(id)
	if err != nil {
		return "", err
	}
	if t.Owner != "" {
		return "", fmt.Errorf("task %d already owned by %s", id, t.Owner)
	}
	t.Owner = owner
	t.Status = "in_progress"
	tm.save(t)
	out, _ := json.MarshalIndent(t, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) ListAll() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	entries, _ := os.ReadDir(tm.dir)
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "task_") && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "No tasks."
	}
	var lines []string
	for _, name := range names {
		data, _ := os.ReadFile(filepath.Join(tm.dir, name))
		var t Task
		if json.Unmarshal(data, &t) != nil {
			continue
		}
		marker := map[string]string{"pending": "[ ]", "in_progress": "[>]", "completed": "[x]"}[t.Status]
		if marker == "" {
			marker = "[?]"
		}
		ownerStr := ""
		if t.Owner != "" {
			ownerStr = " @" + t.Owner
		}
		lines = append(lines, fmt.Sprintf("%s #%d: %s%s", marker, t.ID, t.Subject, ownerStr))
	}
	return strings.Join(lines, "\n")
}

// scanUnclaimedTasks returns tasks with status=pending, owner="", no blockers.
func (tm *TaskManager) scanUnclaimedTasks() []*Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	entries, _ := os.ReadDir(tm.dir)
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "task_") && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var unclaimed []*Task
	for _, name := range names {
		data, _ := os.ReadFile(filepath.Join(tm.dir, name))
		var t Task
		if json.Unmarshal(data, &t) != nil {
			continue
		}
		if t.Status == "pending" && t.Owner == "" && len(t.BlockedBy) == 0 {
			unclaimed = append(unclaimed, &t)
		}
	}
	return unclaimed
}

// ========== Globals ==========

var taskMgr *TaskManager

// ========== Input types for FunctionTools ==========

type TaskCreateInput struct {
	Subject     string `json:"subject" description:"Short task title"`
	Description string `json:"description,omitempty" description:"Optional longer description"`
}

type TaskUpdateInput struct {
	TaskID int    `json:"task_id" description:"Task ID to update"`
	Status string `json:"status,omitempty" description:"New status: pending, in_progress, completed"`
	Owner  string `json:"owner,omitempty" description:"Owner name"`
}

type TaskGetInput struct {
	TaskID int `json:"task_id" description:"Task ID to retrieve"`
}

type TaskClaimInput struct {
	TaskID int    `json:"task_id" description:"Task ID to claim"`
	Owner  string `json:"owner" description:"Name of the claimant"`
}

type TaskListInput struct{}

// ========== Tool implementations ==========

func handleTaskCreate(_ context.Context, in TaskCreateInput) (string, error) {
	result := taskMgr.Create(in.Subject, in.Description, nil)
	return result, nil
}

func handleTaskUpdate(_ context.Context, in TaskUpdateInput) (string, error) {
	result, err := taskMgr.Update(in.TaskID, in.Status, in.Owner)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleTaskGet(_ context.Context, in TaskGetInput) (string, error) {
	result, err := taskMgr.Get(in.TaskID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleTaskClaim(_ context.Context, in TaskClaimInput) (string, error) {
	result, err := taskMgr.Claim(in.TaskID, in.Owner)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleTaskList(_ context.Context, _ TaskListInput) (string, error) {
	return taskMgr.ListAll(), nil
}

// ========== Bash tool ==========

type BashInput struct {
	Command string `json:"command" description:"Shell command to execute"`
}

func handleBash(_ context.Context, in BashInput) (string, error) {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(in.Command, d) {
			return "Error: Dangerous command blocked", nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", in.Command)
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

// ========== Autonomous worker loop ==========

// startWorker launches a background goroutine that polls for unclaimed tasks
// and runs the worker agent to claim and complete them.
func startWorker(m *openai.Model, workerTools []tool.Tool) {
	go func() {
		sess := inmemory.NewSessionService()
		workerAgent := llmagent.New("worker",
			llmagent.WithModel(m),
			llmagent.WithInstruction(fmt.Sprintf(
				"You are an autonomous worker agent at %s. "+
					"When given a task, claim it with task_claim, then do the work using bash and file tools, "+
					"then mark it completed with task_update. Be thorough but concise.", workdir)),
			llmagent.WithTools(workerTools),
			llmagent.WithGenerationConfig(model.GenerationConfig{
				MaxTokens: model.IntPtr(8000),
				Stream:    true,
			}),
			llmagent.WithMaxToolIterations(30),
			llmagent.WithMessageFilterMode(llmagent.IsolatedRequest),
		)
		workerRunner := runner.NewRunner("worker-runner", workerAgent, runner.WithSessionService(sess))
		defer workerRunner.Close()

		sessionSeq := 0
		for {
			time.Sleep(5 * time.Second)
			unclaimed := taskMgr.scanUnclaimedTasks()
			if len(unclaimed) == 0 {
				continue
			}

			task := unclaimed[0]
			sessionSeq++
			sessionID := fmt.Sprintf("worker-session-%d", sessionSeq)

			prompt := fmt.Sprintf(
				"Task #%d: %s\n%s\n\nClaim this task (task_id=%d, owner=\"worker\"), do the work, then mark it completed.",
				task.ID, task.Subject, task.Description, task.ID,
			)

			fmt.Printf("\n  \033[35m[worker]\033[0m Claiming task #%d: %s\n", task.ID, task.Subject)

			evCh, err := workerRunner.Run(context.Background(), "worker", sessionID, model.NewUserMessage(prompt))
			if err != nil {
				fmt.Printf("  \033[35m[worker]\033[0m Error: %v\n", err)
				continue
			}

			// Drain events silently (worker output goes to stdout with worker prefix)
			for ev := range evCh {
				if ev.IsRunnerCompletion() {
					break
				}
				if ev.Response == nil {
					continue
				}
				for _, choice := range ev.Response.Choices {
					if choice.Delta.Content != "" {
						fmt.Printf("\033[35m[worker]\033[0m %s", choice.Delta.Content)
					}
					for _, tc := range choice.Delta.ToolCalls {
						if tc.Function.Name != "" {
							fmt.Printf("  \033[35m[worker] > %s\033[0m\n", tc.Function.Name)
						}
					}
				}
			}
			fmt.Printf("\n  \033[35m[worker]\033[0m Done with task #%d\n", task.ID)
		}
	}()
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

	// -- Task manager --
	tasksDir := filepath.Join(workdir, ".tasks")
	taskMgr = NewTaskManager(tasksDir)

	// -- Tools for lead agent (creates + queries tasks) --
	leadTools := []tool.Tool{
		function.NewFunctionTool(handleBash,
			function.WithName("bash"),
			function.WithDescription("Run a shell command."),
		),
		function.NewFunctionTool(handleTaskCreate,
			function.WithName("task_create"),
			function.WithDescription("Create a task on the task board."),
		),
		function.NewFunctionTool(handleTaskUpdate,
			function.WithName("task_update"),
			function.WithDescription("Update task status or owner."),
		),
		function.NewFunctionTool(handleTaskGet,
			function.WithName("task_get"),
			function.WithDescription("Get task details by ID."),
		),
		function.NewFunctionTool(handleTaskList,
			function.WithName("task_list"),
			function.WithDescription("List all tasks with status."),
		),
	}

	// -- Tools for worker agent (same as lead + claim) --
	workerTools := []tool.Tool{
		function.NewFunctionTool(handleBash,
			function.WithName("bash"),
			function.WithDescription("Run a shell command."),
		),
		function.NewFunctionTool(handleTaskCreate,
			function.WithName("task_create"),
			function.WithDescription("Create a task on the task board."),
		),
		function.NewFunctionTool(handleTaskUpdate,
			function.WithName("task_update"),
			function.WithDescription("Update task status or owner."),
		),
		function.NewFunctionTool(handleTaskGet,
			function.WithName("task_get"),
			function.WithDescription("Get task details by ID."),
		),
		function.NewFunctionTool(handleTaskList,
			function.WithName("task_list"),
			function.WithDescription("List all tasks with status."),
		),
		function.NewFunctionTool(handleTaskClaim,
			function.WithName("task_claim"),
			function.WithDescription("Claim a task by ID and set owner."),
		),
	}

	// -- Lead agent --
	leadAgent := llmagent.New("lead",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are a team lead agent at %s. "+
				"Use task_create to add tasks to the board. "+
				"A background worker automatically picks up unclaimed tasks. "+
				"Use task_list to monitor progress.", workdir)),
		llmagent.WithTools(leadTools),
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

	// -- Start autonomous worker --
	startWorker(m, workerTools)

	// -- REPL --
	fmt.Println("\n  s11: Autonomous Agents (trpc-agent-go)")
	fmt.Print("  \"The agent finds work itself. The task board is the coordination point.\"\n\n")
	fmt.Printf("  Tasks directory: %s\n", tasksDir)
	fmt.Println("  Background worker polls every 5s for unclaimed tasks.")
	fmt.Println("  Local commands: /tasks")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms11 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms11 >> \033[0m")
		}

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "q" || input == "exit" {
			break
		}

		// Local commands
		if input == "/tasks" {
			fmt.Println(taskMgr.ListAll())
			fmt.Println()
			continue
		}

		if guidedIdx < len(guided) {
			query := input
			if query == "" {
				query = guided[guidedIdx].prompt
			}
			guidedIdx++
			evCh, err := r.Run(context.Background(), "user", sessionID, model.NewUserMessage(query))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(evCh)
			if text != "" {
				fmt.Println(text)
			}
		} else {
			if input == "" {
				continue
			}
			evCh, err := r.Run(context.Background(), "user", sessionID, model.NewUserMessage(input))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(evCh)
			if text != "" {
				fmt.Println(text)
			}
		}
		fmt.Println()
	}
}
