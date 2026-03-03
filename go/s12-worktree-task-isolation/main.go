// s12 - Worktree + Task Isolation
//
// Directory-level isolation for parallel task execution.
// Tasks are the control plane; worktrees are the execution plane.
//
//	Control plane (.tasks/)          Execution plane (.worktrees/)
//	+------------------+            +------------------------+
//	| task_1.json      |            | auth-refactor/         |
//	|  status: in_prog |<------>    |  branch: wt/auth-refac |
//	|  worktree: "auth" |           |  task_id: 1            |
//	+------------------+            +------------------------+
//	| task_2.json      |            | ui-login/              |
//	|  worktree: "ui"  |<------>    |  branch: wt/ui-login   |
//	+------------------+            +------------------------+
//	                                 index.json (registry)
//	                                 events.jsonl (lifecycle)
//
// Key insight: "Isolate by directory, coordinate by task ID."
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== EventBus: append-only JSONL lifecycle events ==========

type EventBus struct {
	path string
	mu   sync.Mutex
}

func NewEventBus(path string) *EventBus {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(""), 0o644)
	}
	return &EventBus{path: path}
}

type eventPayload struct {
	Event    string      `json:"event"`
	Ts       float64     `json:"ts"`
	Task     interface{} `json:"task"`
	Worktree interface{} `json:"worktree"`
	Error    string      `json:"error,omitempty"`
}

func (eb *EventBus) Emit(event string, task, worktree interface{}, errMsg string) {
	if task == nil {
		task = map[string]interface{}{}
	}
	if worktree == nil {
		worktree = map[string]interface{}{}
	}
	payload := eventPayload{
		Event:    event,
		Ts:       float64(time.Now().UnixMilli()) / 1000.0,
		Task:     task,
		Worktree: worktree,
	}
	if errMsg != "" {
		payload.Error = errMsg
	}
	data, _ := json.Marshal(payload)
	eb.mu.Lock()
	defer eb.mu.Unlock()
	f, err := os.OpenFile(eb.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

func (eb *EventBus) ListRecent(limit int) string {
	if limit <= 0 {
		limit = 20
	}
	limit = int(math.Min(float64(limit), 200))
	eb.mu.Lock()
	defer eb.mu.Unlock()
	data, err := os.ReadFile(eb.path)
	if err != nil {
		return "[]"
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if lines[0] == "" {
		lines = nil
	}
	start := 0
	if len(lines) > limit {
		start = len(lines) - limit
	}
	recent := lines[start:]
	var items []interface{}
	for _, line := range recent {
		var obj interface{}
		if json.Unmarshal([]byte(line), &obj) == nil {
			items = append(items, obj)
		} else {
			items = append(items, map[string]interface{}{"event": "parse_error", "raw": line})
		}
	}
	out, _ := json.MarshalIndent(items, "", "  ")
	return string(out)
}

// ========== TaskManager: persistent task board with worktree binding ==========

type TaskManager struct {
	dir    string
	nextID int
	mu     sync.Mutex
}

type taskData struct {
	ID          int     `json:"id"`
	Subject     string  `json:"subject"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Owner       string  `json:"owner"`
	Worktree    string  `json:"worktree"`
	BlockedBy   []int   `json:"blockedBy"`
	CreatedAt   float64 `json:"created_at"`
	UpdatedAt   float64 `json:"updated_at"`
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
			var id int
			if _, err := fmt.Sscanf(name, "task_%d.json", &id); err == nil && id > maxVal {
				maxVal = id
			}
		}
	}
	return maxVal
}

func (tm *TaskManager) taskPath(id int) string {
	return filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", id))
}

func (tm *TaskManager) load(id int) (*taskData, error) {
	data, err := os.ReadFile(tm.taskPath(id))
	if err != nil {
		return nil, fmt.Errorf("Task %d not found", id)
	}
	var task taskData
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("corrupt task %d", id)
	}
	return &task, nil
}

func (tm *TaskManager) save(task *taskData) {
	data, _ := json.MarshalIndent(task, "", "  ")
	_ = os.WriteFile(tm.taskPath(task.ID), data, 0o644)
}

func (tm *TaskManager) Exists(id int) bool {
	_, err := os.Stat(tm.taskPath(id))
	return err == nil
}

func (tm *TaskManager) Create(subject, description string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	now := float64(time.Now().UnixMilli()) / 1000.0
	task := &taskData{
		ID:          tm.nextID,
		Subject:     subject,
		Description: description,
		Status:      "pending",
		Owner:       "",
		Worktree:    "",
		BlockedBy:   []int{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tm.save(task)
	tm.nextID++
	out, _ := json.MarshalIndent(task, "", "  ")
	return string(out)
}

func (tm *TaskManager) Get(id int) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, err := tm.load(id)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(task, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) Update(id int, status, owner string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, err := tm.load(id)
	if err != nil {
		return "", err
	}
	if status != "" {
		valid := map[string]bool{"pending": true, "in_progress": true, "completed": true}
		if !valid[status] {
			return "", fmt.Errorf("Invalid status: %s", status)
		}
		task.Status = status
	}
	if owner != "" {
		task.Owner = owner
	}
	task.UpdatedAt = float64(time.Now().UnixMilli()) / 1000.0
	tm.save(task)
	out, _ := json.MarshalIndent(task, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) BindWorktree(id int, worktree, owner string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, err := tm.load(id)
	if err != nil {
		return "", err
	}
	task.Worktree = worktree
	if owner != "" {
		task.Owner = owner
	}
	if task.Status == "pending" {
		task.Status = "in_progress"
	}
	task.UpdatedAt = float64(time.Now().UnixMilli()) / 1000.0
	tm.save(task)
	out, _ := json.MarshalIndent(task, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) UnbindWorktree(id int) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task, err := tm.load(id)
	if err != nil {
		return "", err
	}
	task.Worktree = ""
	task.UpdatedAt = float64(time.Now().UnixMilli()) / 1000.0
	tm.save(task)
	out, _ := json.MarshalIndent(task, "", "  ")
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
		data, err := os.ReadFile(filepath.Join(tm.dir, name))
		if err != nil {
			continue
		}
		var t taskData
		if json.Unmarshal(data, &t) != nil {
			continue
		}
		marker := map[string]string{"pending": "[ ]", "in_progress": "[>]", "completed": "[x]"}[t.Status]
		if marker == "" {
			marker = "[?]"
		}
		ownerStr := ""
		if t.Owner != "" {
			ownerStr = " owner=" + t.Owner
		}
		wtStr := ""
		if t.Worktree != "" {
			wtStr = " wt=" + t.Worktree
		}
		lines = append(lines, fmt.Sprintf("%s #%d: %s%s%s", marker, t.ID, t.Subject, ownerStr, wtStr))
	}
	return strings.Join(lines, "\n")
}

// ========== WorktreeManager: git worktree lifecycle + index ==========

type worktreeEntry struct {
	Name      string  `json:"name"`
	Path      string  `json:"path"`
	Branch    string  `json:"branch"`
	TaskID    *int    `json:"task_id"`
	Status    string  `json:"status"`
	CreatedAt float64 `json:"created_at,omitempty"`
	RemovedAt float64 `json:"removed_at,omitempty"`
	KeptAt    float64 `json:"kept_at,omitempty"`
}

type worktreeIndex struct {
	Worktrees []worktreeEntry `json:"worktrees"`
}

type WorktreeManager struct {
	repoRoot     string
	dir          string
	indexPath    string
	tasks        *TaskManager
	events       *EventBus
	gitAvailable bool
	mu           sync.Mutex
}

var nameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{1,40}$`)

func NewWorktreeManager(repoRoot string, tasks *TaskManager, events *EventBus) *WorktreeManager {
	dir := filepath.Join(repoRoot, ".worktrees")
	_ = os.MkdirAll(dir, 0o755)
	indexPath := filepath.Join(dir, "index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		data, _ := json.MarshalIndent(worktreeIndex{Worktrees: []worktreeEntry{}}, "", "  ")
		_ = os.WriteFile(indexPath, data, 0o644)
	}
	wm := &WorktreeManager{
		repoRoot:  repoRoot,
		dir:       dir,
		indexPath: indexPath,
		tasks:     tasks,
		events:    events,
	}
	wm.gitAvailable = wm.isGitRepo()
	return wm
}

func (wm *WorktreeManager) isGitRepo() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = wm.repoRoot
	output, err := cmd.CombinedOutput()
	_ = output
	return err == nil
}

func (wm *WorktreeManager) runGit(args ...string) (string, error) {
	if !wm.gitAvailable {
		return "", fmt.Errorf("Not in a git repository. worktree tools require git.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = wm.repoRoot
	output, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(output))
	if err != nil {
		if out != "" {
			return "", fmt.Errorf("%s", out)
		}
		return "", fmt.Errorf("git %s failed", strings.Join(args, " "))
	}
	if out == "" {
		return "(no output)", nil
	}
	return out, nil
}

func (wm *WorktreeManager) loadIndex() worktreeIndex {
	data, err := os.ReadFile(wm.indexPath)
	if err != nil {
		return worktreeIndex{Worktrees: []worktreeEntry{}}
	}
	var idx worktreeIndex
	if json.Unmarshal(data, &idx) != nil {
		return worktreeIndex{Worktrees: []worktreeEntry{}}
	}
	return idx
}

func (wm *WorktreeManager) saveIndex(idx worktreeIndex) {
	data, _ := json.MarshalIndent(idx, "", "  ")
	_ = os.WriteFile(wm.indexPath, data, 0o644)
}

func (wm *WorktreeManager) find(name string) *worktreeEntry {
	idx := wm.loadIndex()
	for i := range idx.Worktrees {
		if idx.Worktrees[i].Name == name {
			return &idx.Worktrees[i]
		}
	}
	return nil
}

func (wm *WorktreeManager) validateName(name string) error {
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("Invalid worktree name. Use 1-40 chars: letters, numbers, ., _, -")
	}
	return nil
}

func (wm *WorktreeManager) Create(name string, taskID *int, baseRef string) (string, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if err := wm.validateName(name); err != nil {
		return "", err
	}
	if wm.find(name) != nil {
		return "", fmt.Errorf("Worktree '%s' already exists in index", name)
	}
	if taskID != nil && !wm.tasks.Exists(*taskID) {
		return "", fmt.Errorf("Task %d not found", *taskID)
	}
	if baseRef == "" {
		baseRef = "HEAD"
	}

	// Emit before event
	taskInfo := map[string]interface{}{}
	if taskID != nil {
		taskInfo["id"] = *taskID
	}
	wm.events.Emit("worktree.create.before", taskInfo,
		map[string]interface{}{"name": name, "base_ref": baseRef}, "")

	path := filepath.Join(wm.dir, name)
	branch := "wt/" + name

	_, err := wm.runGit("worktree", "add", "-b", branch, path, baseRef)
	if err != nil {
		wm.events.Emit("worktree.create.failed", taskInfo,
			map[string]interface{}{"name": name, "base_ref": baseRef}, err.Error())
		return "", err
	}

	entry := worktreeEntry{
		Name:      name,
		Path:      path,
		Branch:    branch,
		TaskID:    taskID,
		Status:    "active",
		CreatedAt: float64(time.Now().UnixMilli()) / 1000.0,
	}

	idx := wm.loadIndex()
	idx.Worktrees = append(idx.Worktrees, entry)
	wm.saveIndex(idx)

	if taskID != nil {
		wm.tasks.BindWorktree(*taskID, name, "")
	}

	wm.events.Emit("worktree.create.after", taskInfo,
		map[string]interface{}{"name": name, "path": path, "branch": branch, "status": "active"}, "")

	out, _ := json.MarshalIndent(entry, "", "  ")
	return string(out), nil
}

func (wm *WorktreeManager) ListAll() string {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	idx := wm.loadIndex()
	if len(idx.Worktrees) == 0 {
		return "No worktrees in index."
	}
	var lines []string
	for _, wt := range idx.Worktrees {
		suffix := ""
		if wt.TaskID != nil {
			suffix = fmt.Sprintf(" task=%d", *wt.TaskID)
		}
		lines = append(lines, fmt.Sprintf("[%s] %s -> %s (%s)%s",
			wt.Status, wt.Name, wt.Path, wt.Branch, suffix))
	}
	return strings.Join(lines, "\n")
}

func (wm *WorktreeManager) Status(name string) string {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wt := wm.find(name)
	if wt == nil {
		return fmt.Sprintf("Error: Unknown worktree '%s'", name)
	}
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		return fmt.Sprintf("Error: Worktree path missing: %s", wt.Path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "status", "--short", "--branch")
	cmd.Dir = wt.Path
	output, _ := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if text == "" {
		return "Clean worktree"
	}
	return text
}

func (wm *WorktreeManager) Run(name, command string) string {
	wm.mu.Lock()
	wt := wm.find(name)
	wm.mu.Unlock()

	if wt == nil {
		return fmt.Sprintf("Error: Unknown worktree '%s'", name)
	}
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		return fmt.Sprintf("Error: Worktree path missing: %s", wt.Path)
	}

	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = wt.Path
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (300s)"
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

func (wm *WorktreeManager) Remove(name string, force, completeTask bool) (string, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wt := wm.find(name)
	if wt == nil {
		return "", fmt.Errorf("Unknown worktree '%s'", name)
	}

	taskInfo := map[string]interface{}{}
	if wt.TaskID != nil {
		taskInfo["id"] = *wt.TaskID
	}
	wm.events.Emit("worktree.remove.before", taskInfo,
		map[string]interface{}{"name": name, "path": wt.Path}, "")

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, wt.Path)

	_, err := wm.runGit(args...)
	if err != nil {
		wm.events.Emit("worktree.remove.failed", taskInfo,
			map[string]interface{}{"name": name, "path": wt.Path}, err.Error())
		return "", err
	}

	// Complete bound task if requested
	if completeTask && wt.TaskID != nil {
		taskID := *wt.TaskID
		taskJSON, _ := wm.tasks.Get(taskID)
		var before taskData
		_ = json.Unmarshal([]byte(taskJSON), &before)
		wm.tasks.Update(taskID, "completed", "")
		wm.tasks.UnbindWorktree(taskID)
		wm.events.Emit("task.completed",
			map[string]interface{}{"id": taskID, "subject": before.Subject, "status": "completed"},
			map[string]interface{}{"name": name}, "")
	}

	// Update index
	idx := wm.loadIndex()
	now := float64(time.Now().UnixMilli()) / 1000.0
	for i := range idx.Worktrees {
		if idx.Worktrees[i].Name == name {
			idx.Worktrees[i].Status = "removed"
			idx.Worktrees[i].RemovedAt = now
		}
	}
	wm.saveIndex(idx)

	wm.events.Emit("worktree.remove.after", taskInfo,
		map[string]interface{}{"name": name, "path": wt.Path, "status": "removed"}, "")

	return fmt.Sprintf("Removed worktree '%s'", name), nil
}

func (wm *WorktreeManager) Keep(name string) (string, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wt := wm.find(name)
	if wt == nil {
		return "", fmt.Errorf("Unknown worktree '%s'", name)
	}

	idx := wm.loadIndex()
	now := float64(time.Now().UnixMilli()) / 1000.0
	var kept *worktreeEntry
	for i := range idx.Worktrees {
		if idx.Worktrees[i].Name == name {
			idx.Worktrees[i].Status = "kept"
			idx.Worktrees[i].KeptAt = now
			kept = &idx.Worktrees[i]
		}
	}
	wm.saveIndex(idx)

	taskInfo := map[string]interface{}{}
	if wt.TaskID != nil {
		taskInfo["id"] = *wt.TaskID
	}
	wm.events.Emit("worktree.keep", taskInfo,
		map[string]interface{}{"name": name, "path": wt.Path, "status": "kept"}, "")

	if kept != nil {
		out, _ := json.MarshalIndent(kept, "", "  ")
		return string(out), nil
	}
	return fmt.Sprintf("Kept worktree '%s'", name), nil
}

// ========== Globals ==========

var (
	modelID   string
	systemMsg string
	workDir   string
	repoRoot  string
	tasks     *TaskManager
	events    *EventBus
	worktrees *WorktreeManager
	apiClient anthropic.Client
	tools     []anthropic.ToolUnionParam
	dispatch  map[string]func(json.RawMessage) string
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Create tasks", `Create tasks for "backend auth module" and "frontend login page", then list tasks.`},
	{"Bind worktrees to tasks", `Create worktree "auth-refactor" for task 1, then bind task 2 to a new worktree "ui-login".`},
	{"Run commands in worktree", `Run "git status --short" in worktree "auth-refactor".`},
	{"Manage worktree lifecycle", `Keep worktree "ui-login", then list worktrees and inspect events.`},
	{"Clean up with task completion", `Remove worktree "auth-refactor" with complete_task=true, then list tasks/worktrees/events.`},
}

// ========== Path safety + base tool implementations ==========

func safePath(p string) (string, error) {
	full := filepath.Join(workDir, p)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", p)
	}
	if abs != workDir && !strings.HasPrefix(abs, workDir+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
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
	cmd.Dir = workDir
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

func runWrite(path, content string) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	_ = os.MkdirAll(filepath.Dir(fp), 0o755)
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Wrote %d bytes", len(content))
}

func runEdit(path, oldText, newText string) string {
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
	if err := os.WriteFile(fp, []byte(strings.Replace(content, oldText, newText, 1)), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Edited %s", path)
}

// ========== detect_repo_root ==========

func detectRepoRoot(cwd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		return cwd
	}
	root := strings.TrimSpace(string(output))
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		return root
	}
	return cwd
}

// ========== Tool definitions (16 tools) ==========

func buildTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		// 4 base tools
		{OfTool: &anthropic.ToolParam{Name: "bash", Description: anthropic.String("Run a shell command in the current workspace (blocking)."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"command": map[string]interface{}{"type": "string"}},
				Required:   []string{"command"}}}},
		{OfTool: &anthropic.ToolParam{Name: "read_file", Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "integer"}},
				Required:   []string{"path"}}}},
		{OfTool: &anthropic.ToolParam{Name: "write_file", Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}},
				Required:   []string{"path", "content"}}}},
		{OfTool: &anthropic.ToolParam{Name: "edit_file", Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "old_text": map[string]interface{}{"type": "string"}, "new_text": map[string]interface{}{"type": "string"}},
				Required:   []string{"path", "old_text", "new_text"}}}},
		// 5 task tools
		{OfTool: &anthropic.ToolParam{Name: "task_create", Description: anthropic.String("Create a new task on the shared task board."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"subject": map[string]interface{}{"type": "string"}, "description": map[string]interface{}{"type": "string"}},
				Required:   []string{"subject"}}}},
		{OfTool: &anthropic.ToolParam{Name: "task_list", Description: anthropic.String("List all tasks with status, owner, and worktree binding."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "task_get", Description: anthropic.String("Get task details by ID."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"task_id": map[string]interface{}{"type": "integer"}},
				Required:   []string{"task_id"}}}},
		{OfTool: &anthropic.ToolParam{Name: "task_update", Description: anthropic.String("Update task status or owner."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{"type": "integer"},
					"status":  map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
					"owner":   map[string]interface{}{"type": "string"}},
				Required: []string{"task_id"}}}},
		{OfTool: &anthropic.ToolParam{Name: "task_bind_worktree", Description: anthropic.String("Bind a task to a worktree name."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"task_id":  map[string]interface{}{"type": "integer"},
					"worktree": map[string]interface{}{"type": "string"},
					"owner":    map[string]interface{}{"type": "string"}},
				Required: []string{"task_id", "worktree"}}}},
		// 7 worktree tools
		{OfTool: &anthropic.ToolParam{Name: "worktree_create", Description: anthropic.String("Create a git worktree and optionally bind it to a task."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"name":     map[string]interface{}{"type": "string"},
					"task_id":  map[string]interface{}{"type": "integer"},
					"base_ref": map[string]interface{}{"type": "string"}},
				Required: []string{"name"}}}},
		{OfTool: &anthropic.ToolParam{Name: "worktree_list", Description: anthropic.String("List worktrees tracked in .worktrees/index.json."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "worktree_status", Description: anthropic.String("Show git status for one worktree."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"name": map[string]interface{}{"type": "string"}},
				Required:   []string{"name"}}}},
		{OfTool: &anthropic.ToolParam{Name: "worktree_run", Description: anthropic.String("Run a shell command in a named worktree directory."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "command": map[string]interface{}{"type": "string"}},
				Required:   []string{"name", "command"}}}},
		{OfTool: &anthropic.ToolParam{Name: "worktree_remove", Description: anthropic.String("Remove a worktree and optionally mark its bound task completed."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"name":          map[string]interface{}{"type": "string"},
					"force":         map[string]interface{}{"type": "boolean"},
					"complete_task": map[string]interface{}{"type": "boolean"}},
				Required: []string{"name"}}}},
		{OfTool: &anthropic.ToolParam{Name: "worktree_keep", Description: anthropic.String("Mark a worktree as kept in lifecycle state without removing it."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"name": map[string]interface{}{"type": "string"}},
				Required:   []string{"name"}}}},
		{OfTool: &anthropic.ToolParam{Name: "worktree_events", Description: anthropic.String("List recent worktree/task lifecycle events from .worktrees/events.jsonl."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"limit": map[string]interface{}{"type": "integer"}}}}},
	}
}

// ========== Dispatch map (16 handlers) ==========

func buildDispatch() map[string]func(json.RawMessage) string {
	return map[string]func(json.RawMessage) string{
		// 4 base tools
		"bash": func(raw json.RawMessage) string {
			var in struct {
				Command string `json:"command"`
			}
			_ = json.Unmarshal(raw, &in)
			return runBash(in.Command)
		},
		"read_file": func(raw json.RawMessage) string {
			var in struct {
				Path  string `json:"path"`
				Limit int    `json:"limit"`
			}
			_ = json.Unmarshal(raw, &in)
			return runRead(in.Path, in.Limit)
		},
		"write_file": func(raw json.RawMessage) string {
			var in struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			_ = json.Unmarshal(raw, &in)
			return runWrite(in.Path, in.Content)
		},
		"edit_file": func(raw json.RawMessage) string {
			var in struct {
				Path    string `json:"path"`
				OldText string `json:"old_text"`
				NewText string `json:"new_text"`
			}
			_ = json.Unmarshal(raw, &in)
			return runEdit(in.Path, in.OldText, in.NewText)
		},
		// 5 task tools
		"task_create": func(raw json.RawMessage) string {
			var in struct {
				Subject     string `json:"subject"`
				Description string `json:"description"`
			}
			_ = json.Unmarshal(raw, &in)
			return tasks.Create(in.Subject, in.Description)
		},
		"task_list": func(raw json.RawMessage) string {
			return tasks.ListAll()
		},
		"task_get": func(raw json.RawMessage) string {
			var in struct {
				TaskID int `json:"task_id"`
			}
			_ = json.Unmarshal(raw, &in)
			result, err := tasks.Get(in.TaskID)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return result
		},
		"task_update": func(raw json.RawMessage) string {
			var in struct {
				TaskID int    `json:"task_id"`
				Status string `json:"status"`
				Owner  string `json:"owner"`
			}
			_ = json.Unmarshal(raw, &in)
			result, err := tasks.Update(in.TaskID, in.Status, in.Owner)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return result
		},
		"task_bind_worktree": func(raw json.RawMessage) string {
			var in struct {
				TaskID   int    `json:"task_id"`
				Worktree string `json:"worktree"`
				Owner    string `json:"owner"`
			}
			_ = json.Unmarshal(raw, &in)
			result, err := tasks.BindWorktree(in.TaskID, in.Worktree, in.Owner)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return result
		},
		// 7 worktree tools
		"worktree_create": func(raw json.RawMessage) string {
			var in struct {
				Name    string `json:"name"`
				TaskID  *int   `json:"task_id"`
				BaseRef string `json:"base_ref"`
			}
			_ = json.Unmarshal(raw, &in)
			result, err := worktrees.Create(in.Name, in.TaskID, in.BaseRef)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return result
		},
		"worktree_list": func(raw json.RawMessage) string {
			return worktrees.ListAll()
		},
		"worktree_status": func(raw json.RawMessage) string {
			var in struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(raw, &in)
			return worktrees.Status(in.Name)
		},
		"worktree_run": func(raw json.RawMessage) string {
			var in struct {
				Name    string `json:"name"`
				Command string `json:"command"`
			}
			_ = json.Unmarshal(raw, &in)
			return worktrees.Run(in.Name, in.Command)
		},
		"worktree_remove": func(raw json.RawMessage) string {
			var in struct {
				Name         string `json:"name"`
				Force        bool   `json:"force"`
				CompleteTask bool   `json:"complete_task"`
			}
			_ = json.Unmarshal(raw, &in)
			result, err := worktrees.Remove(in.Name, in.Force, in.CompleteTask)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return result
		},
		"worktree_keep": func(raw json.RawMessage) string {
			var in struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(raw, &in)
			result, err := worktrees.Keep(in.Name)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			return result
		},
		"worktree_events": func(raw json.RawMessage) string {
			var in struct {
				Limit int `json:"limit"`
			}
			_ = json.Unmarshal(raw, &in)
			return events.ListRecent(in.Limit)
		},
	}
}

// ========== Agent Loop ==========

func agentLoop(messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		resp, err := apiClient.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(modelID),
			System:    []anthropic.TextBlockParam{{Text: systemMsg}},
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 8000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "API error: %v\n", err)
			return messages
		}

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
	modelID = os.Getenv("MODEL_ID")
	if modelID == "" {
		modelID = "claude-sonnet-4-20250514"
	}
	workDir, _ = os.Getwd()
	repoRoot = detectRepoRoot(workDir)

	opts := []option.RequestOption{}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	apiClient = anthropic.NewClient(opts...)

	tasks = NewTaskManager(filepath.Join(repoRoot, ".tasks"))
	events = NewEventBus(filepath.Join(repoRoot, ".worktrees", "events.jsonl"))
	worktrees = NewWorktreeManager(repoRoot, tasks, events)

	systemMsg = fmt.Sprintf(
		"You are a coding agent at %s. "+
			"Use task + worktree tools for multi-task work. "+
			"For parallel or risky changes: create tasks, allocate worktree lanes, "+
			"run commands in those lanes, then choose keep/remove for closeout. "+
			"Use worktree_events when you need lifecycle visibility.",
		workDir,
	)

	tools = buildTools()
	dispatch = buildDispatch()

	fmt.Printf("Repo root for s12: %s\n", repoRoot)
	if !worktrees.gitAvailable {
		fmt.Println("Note: Not in a git repo. worktree_* tools will return errors.")
	}

	fmt.Println("\n  s12: Worktree + Task Isolation")
	fmt.Print("  \"Isolated workspaces for parallel tasks\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		var query string

		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms12 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
			if !scanner.Scan() {
				break
			}
			query = strings.TrimSpace(scanner.Text())
			if query == "q" || query == "exit" {
				break
			}
			// Local commands: handle without advancing guidedIdx
			if strings.HasPrefix(query, "/") {
				if query == "/tasks" {
					fmt.Println(tasks.ListAll())
				} else if query == "/worktrees" {
					fmt.Println(worktrees.ListAll())
				} else if query == "/events" {
					fmt.Println(events.ListRecent(20))
				}
				continue
			}
			// Empty input: use guided prompt
			if query == "" {
				query = guided[guidedIdx].prompt
			}
			guidedIdx++
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Printf("\033[36ms12 >> \033[0m")
			if !scanner.Scan() {
				break
			}
			query = strings.TrimSpace(scanner.Text())
			if query == "q" || query == "exit" {
				break
			}
			if query == "" {
				continue
			}
			// Local commands
			if query == "/tasks" {
				fmt.Println(tasks.ListAll())
				continue
			}
			if query == "/worktrees" {
				fmt.Println(worktrees.ListAll())
				continue
			}
			if query == "/events" {
				fmt.Println(events.ListRecent(20))
				continue
			}
		}

		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		messages = agentLoop(messages)
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
