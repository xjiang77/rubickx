// s12 - Worktree + Task Isolation (trpc-agent-go)
//
// Git worktrees provide directory-level isolation for parallel task execution.
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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

var workDir string

var guided = []struct {
	step   string
	prompt string
}{
	{"Create tasks and worktrees", `Create tasks for "backend auth module" and "frontend login page", then create worktrees "auth-refactor" for task 1 and "ui-login" for task 2`},
	{"Run commands in isolation", `Run "ls -la" in worktree "auth-refactor" to see the isolated workspace`},
	{"Manage worktree lifecycle", `List all worktrees and events, then keep worktree "ui-login"`},
	{"Complete and clean up", `Remove worktree "auth-refactor" with complete_task=true, then list tasks and events`},
}

// ========== EventBus ==========

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

func (eb *EventBus) Emit(evName string, taskInfo, worktreeInfo interface{}, errMsg string) {
	if taskInfo == nil {
		taskInfo = map[string]interface{}{}
	}
	if worktreeInfo == nil {
		worktreeInfo = map[string]interface{}{}
	}
	payload := eventPayload{
		Event:    evName,
		Ts:       float64(time.Now().UnixMilli()) / 1000.0,
		Task:     taskInfo,
		Worktree: worktreeInfo,
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
	if limit > 200 {
		limit = 200
	}
	eb.mu.Lock()
	defer eb.mu.Unlock()
	data, err := os.ReadFile(eb.path)
	if err != nil {
		return "[]"
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "[]"
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
		}
	}
	out, _ := json.MarshalIndent(items, "", "  ")
	return string(out)
}

// ========== TaskManager ==========

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
		return nil, fmt.Errorf("task %d not found", id)
	}
	var t taskData
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("corrupt task %d", id)
	}
	return &t, nil
}

func (tm *TaskManager) save(t *taskData) {
	data, _ := json.MarshalIndent(t, "", "  ")
	_ = os.WriteFile(tm.taskPath(t.ID), data, 0o644)
}

func (tm *TaskManager) Exists(id int) bool {
	_, err := os.Stat(tm.taskPath(id))
	return err == nil
}

func (tm *TaskManager) Create(subject, description string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	now := float64(time.Now().UnixMilli()) / 1000.0
	t := &taskData{
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
	t.UpdatedAt = float64(time.Now().UnixMilli()) / 1000.0
	tm.save(t)
	out, _ := json.MarshalIndent(t, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) BindWorktree(id int, worktree, owner string) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	t, err := tm.load(id)
	if err != nil {
		return "", err
	}
	t.Worktree = worktree
	if owner != "" {
		t.Owner = owner
	}
	if t.Status == "pending" {
		t.Status = "in_progress"
	}
	t.UpdatedAt = float64(time.Now().UnixMilli()) / 1000.0
	tm.save(t)
	out, _ := json.MarshalIndent(t, "", "  ")
	return string(out), nil
}

func (tm *TaskManager) UnbindWorktree(id int) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	t, err := tm.load(id)
	if err != nil {
		return "", err
	}
	t.Worktree = ""
	t.UpdatedAt = float64(time.Now().UnixMilli()) / 1000.0
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

// ========== WorktreeManager ==========

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
	_, err := cmd.CombinedOutput()
	return err == nil
}

func (wm *WorktreeManager) runGit(args ...string) (string, error) {
	if !wm.gitAvailable {
		return "", fmt.Errorf("not in a git repository; worktree tools require git")
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

func (wm *WorktreeManager) findEntry(name string) *worktreeEntry {
	idx := wm.loadIndex()
	for i := range idx.Worktrees {
		if idx.Worktrees[i].Name == name {
			return &idx.Worktrees[i]
		}
	}
	return nil
}

func (wm *WorktreeManager) Create(name string, taskID *int, baseRef string) (string, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !nameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid worktree name; use 1-40 chars: letters, numbers, ., _, -")
	}
	if wm.findEntry(name) != nil {
		return "", fmt.Errorf("worktree '%s' already exists in index", name)
	}
	if taskID != nil && !wm.tasks.Exists(*taskID) {
		return "", fmt.Errorf("task %d not found", *taskID)
	}
	if baseRef == "" {
		baseRef = "HEAD"
	}

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

func (wm *WorktreeManager) Run(name, command string) string {
	wm.mu.Lock()
	wt := wm.findEntry(name)
	wm.mu.Unlock()

	if wt == nil {
		return fmt.Sprintf("Error: unknown worktree '%s'", name)
	}
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		return fmt.Sprintf("Error: worktree path missing: %s", wt.Path)
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

	wt := wm.findEntry(name)
	if wt == nil {
		return "", fmt.Errorf("unknown worktree '%s'", name)
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

	if completeTask && wt.TaskID != nil {
		taskID := *wt.TaskID
		wm.tasks.Update(taskID, "completed", "")
		wm.tasks.UnbindWorktree(taskID)
		wm.events.Emit("task.completed",
			map[string]interface{}{"id": taskID, "status": "completed"},
			map[string]interface{}{"name": name}, "")
	}

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

	wt := wm.findEntry(name)
	if wt == nil {
		return "", fmt.Errorf("unknown worktree '%s'", name)
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

// ========== detect repo root ==========

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

// ========== Globals ==========

var (
	tasks     *TaskManager
	events    *EventBus
	worktrees *WorktreeManager
)

// ========== Input types for FunctionTools ==========

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
	OldText string `json:"old_text" description:"Exact text to replace"`
	NewText string `json:"new_text" description:"Replacement text"`
}

type TaskCreateInput struct {
	Subject     string `json:"subject" description:"Short task title"`
	Description string `json:"description,omitempty" description:"Longer description"`
}

type TaskListInput struct{}

type TaskGetInput struct {
	TaskID int `json:"task_id" description:"Task ID"`
}

type TaskUpdateInput struct {
	TaskID int    `json:"task_id" description:"Task ID"`
	Status string `json:"status,omitempty" description:"pending, in_progress, completed"`
	Owner  string `json:"owner,omitempty" description:"Owner name"`
}

type WorktreeCreateInput struct {
	Name    string `json:"name" description:"Worktree name (letters, numbers, ., _, -)"`
	TaskID  *int   `json:"task_id,omitempty" description:"Optional task ID to bind"`
	BaseRef string `json:"base_ref,omitempty" description:"Git ref to base on (default: HEAD)"`
}

type WorktreeListInput struct{}

type WorktreeRunInput struct {
	Name    string `json:"name" description:"Worktree name"`
	Command string `json:"command" description:"Shell command to run in the worktree"`
}

type WorktreeRemoveInput struct {
	Name         string `json:"name" description:"Worktree name"`
	Force        bool   `json:"force,omitempty" description:"Force remove even with uncommitted changes"`
	CompleteTask bool   `json:"complete_task,omitempty" description:"Mark bound task as completed on remove"`
}

type WorktreeKeepInput struct {
	Name string `json:"name" description:"Worktree name to mark as kept"`
}

type WorktreeEventsInput struct {
	Limit int `json:"limit,omitempty" description:"Max events to return (default 20)"`
}

// ========== Path safety ==========

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

// ========== Tool implementations ==========

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
	cmd.Dir = workDir
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

func handleReadFile(_ context.Context, in ReadFileInput) (string, error) {
	fp, err := safePath(in.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	lines := strings.Split(string(data), "\n")
	if in.Limit > 0 && in.Limit < len(lines) {
		lines = append(lines[:in.Limit], fmt.Sprintf("... (%d more lines)", len(lines)-in.Limit))
	}
	out := strings.Join(lines, "\n")
	if len(out) > 50000 {
		return out[:50000], nil
	}
	return out, nil
}

func handleWriteFile(_ context.Context, in WriteFileInput) (string, error) {
	fp, err := safePath(in.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if err := os.WriteFile(fp, []byte(in.Content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(in.Content), in.Path), nil
}

func handleEditFile(_ context.Context, in EditFileInput) (string, error) {
	fp, err := safePath(in.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	content := string(data)
	if !strings.Contains(content, in.OldText) {
		return fmt.Sprintf("Error: text not found in %s", in.Path), nil
	}
	newContent := strings.Replace(content, in.OldText, in.NewText, 1)
	if err := os.WriteFile(fp, []byte(newContent), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Edited %s", in.Path), nil
}

func handleTaskCreate(_ context.Context, in TaskCreateInput) (string, error) {
	return tasks.Create(in.Subject, in.Description), nil
}

func handleTaskList(_ context.Context, _ TaskListInput) (string, error) {
	return tasks.ListAll(), nil
}

func handleTaskGet(_ context.Context, in TaskGetInput) (string, error) {
	result, err := tasks.Get(in.TaskID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleTaskUpdate(_ context.Context, in TaskUpdateInput) (string, error) {
	result, err := tasks.Update(in.TaskID, in.Status, in.Owner)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleWorktreeCreate(_ context.Context, in WorktreeCreateInput) (string, error) {
	result, err := worktrees.Create(in.Name, in.TaskID, in.BaseRef)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleWorktreeList(_ context.Context, _ WorktreeListInput) (string, error) {
	return worktrees.ListAll(), nil
}

func handleWorktreeRun(_ context.Context, in WorktreeRunInput) (string, error) {
	return worktrees.Run(in.Name, in.Command), nil
}

func handleWorktreeRemove(_ context.Context, in WorktreeRemoveInput) (string, error) {
	result, err := worktrees.Remove(in.Name, in.Force, in.CompleteTask)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleWorktreeKeep(_ context.Context, in WorktreeKeepInput) (string, error) {
	result, err := worktrees.Keep(in.Name)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return result, nil
}

func handleWorktreeEvents(_ context.Context, in WorktreeEventsInput) (string, error) {
	return events.ListRecent(in.Limit), nil
}

// ========== Event consumer ==========

func consumeEvents(evCh <-chan *event.Event) string {
	var lastText string
	for ev := range evCh {
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
	workDir, _ = os.Getwd()
	repoRoot := detectRepoRoot(workDir)

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

	// -- Managers --
	tasks = NewTaskManager(filepath.Join(repoRoot, ".tasks"))
	events = NewEventBus(filepath.Join(repoRoot, ".worktrees", "events.jsonl"))
	worktrees = NewWorktreeManager(repoRoot, tasks, events)

	// -- Tools --
	allTools := []tool.Tool{
		// Base tools
		function.NewFunctionTool(handleBash,
			function.WithName("bash"),
			function.WithDescription("Run a shell command in the current workspace."),
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
		// Task tools
		function.NewFunctionTool(handleTaskCreate,
			function.WithName("task_create"),
			function.WithDescription("Create a task on the task board."),
		),
		function.NewFunctionTool(handleTaskList,
			function.WithName("task_list"),
			function.WithDescription("List all tasks with status, owner, and worktree."),
		),
		function.NewFunctionTool(handleTaskGet,
			function.WithName("task_get"),
			function.WithDescription("Get task details by ID."),
		),
		function.NewFunctionTool(handleTaskUpdate,
			function.WithName("task_update"),
			function.WithDescription("Update task status or owner."),
		),
		// Worktree tools
		function.NewFunctionTool(handleWorktreeCreate,
			function.WithName("worktree_create"),
			function.WithDescription("Create a git worktree and optionally bind it to a task."),
		),
		function.NewFunctionTool(handleWorktreeList,
			function.WithName("worktree_list"),
			function.WithDescription("List all worktrees from the index."),
		),
		function.NewFunctionTool(handleWorktreeRun,
			function.WithName("worktree_run"),
			function.WithDescription("Run a shell command inside a named worktree directory."),
		),
		function.NewFunctionTool(handleWorktreeRemove,
			function.WithName("worktree_remove"),
			function.WithDescription("Remove a worktree and optionally mark its task completed."),
		),
		function.NewFunctionTool(handleWorktreeKeep,
			function.WithName("worktree_keep"),
			function.WithDescription("Mark a worktree as kept without removing it."),
		),
		function.NewFunctionTool(handleWorktreeEvents,
			function.WithName("worktree_events"),
			function.WithDescription("List recent lifecycle events from events.jsonl."),
		),
	}

	// -- Agent --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(fmt.Sprintf(
			"You are a coding agent at %s (repo root: %s). "+
				"Use task + worktree tools for multi-task parallel work. "+
				"Create tasks, create worktrees (optionally binding a task_id), "+
				"run commands in those worktrees, then keep or remove them. "+
				"Use worktree_events to inspect lifecycle history.",
			workDir, repoRoot)),
		llmagent.WithTools(allTools),
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

	fmt.Printf("  Repo root: %s\n", repoRoot)
	if !worktrees.gitAvailable {
		fmt.Println("  Note: Not in a git repo. worktree_* tools will return errors.")
	}

	// -- REPL --
	fmt.Println("\n  s12: Worktree + Task Isolation (trpc-agent-go)")
	fmt.Print("  \"Isolate by directory, coordinate by task ID.\"\n\n")
	fmt.Println("  Local commands: /tasks, /worktrees, /events")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms12 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms12 >> \033[0m")
		}

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "q" || input == "exit" {
			break
		}

		// Local commands
		switch input {
		case "/tasks":
			fmt.Println(tasks.ListAll())
			fmt.Println()
			continue
		case "/worktrees":
			fmt.Println(worktrees.ListAll())
			fmt.Println()
			continue
		case "/events":
			fmt.Println(events.ListRecent(20))
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
